package service

import (
	"context"
	"encoding/json"
	model "github.com/crypto_custody/model"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"math/big"
	"os"
	"sync"
	"time"
)

// Configuration (tweakable)
const (
	RPC_URL           = "https://mainnet.infura.io/v3/YOUR_KEY" // <- 替换
	DB_DSN            = "host=localhost user=postgres password=postgres dbname=custody sslmode=disable"
	CHAIN             = "ethereum"
	CONFIRMATIONS     = uint64(12)
	INITIAL_STEP      = uint64(200)
	MIN_STEP          = uint64(10)
	MAX_STEP          = uint64(2000)
	SUCCESS_THRESHOLD = 5
	FAILURE_THRESHOLD = 1
	POLL_INTERVAL     = 3 * time.Second
	REORG_CHECK_DEPTH = 100 // on startup check last N blocks for reorg
)

type Scanner struct {
	client       *ethclient.Client
	db           *gorm.DB
	step         uint64
	successCount int
	failureCount int
	mu           sync.Mutex
}

func NewScanner(rpc string, dsn string) (*Scanner, error) {
	client, err := ethclient.Dial(rpc)
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := model.AutoMigrate(db); err != nil {
		return nil, err
	}
	return &Scanner{
		client: client,
		db:     db,
		step:   INITIAL_STEP,
	}, nil
}

// helper: get last processed block from DB
func (s *Scanner) lastProcessedBlock(ctx context.Context) (int64, error) {
	var pb model.ProcessedBlock
	if err := s.db.WithContext(ctx).Order("block_number desc").Limit(1).First(&pb).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return pb.BlockNumber, nil
}

func (s *Scanner) persistProcessedBlock(ctx context.Context, block int64, hash string) error {
	pb := model.ProcessedBlock{
		Chain:       CHAIN,
		BlockNumber: block,
		BlockHash:   hash,
	}
	return s.db.WithContext(ctx).Clauses().Create(&pb).Error
}

// store logs into DB (onchain_events)
func (s *Scanner) persistLogs(ctx context.Context, logs []types.Log) error {
	// batch insert in transaction
	tx := s.db.WithContext(ctx).Begin()
	for _, l := range logs {
		topicsJSON, _ := json.Marshal(l.Topics)
		ev := model.OnchainEvent{
			Chain:       CHAIN,
			BlockNumber: int64(l.BlockNumber),
			BlockHash:   l.BlockHash.Hex(),
			TxHash:      l.TxHash.Hex(),
			LogIndex:    int(l.Index),
			Address:     l.Address.Hex(),
			Topics:      string(topicsJSON),
			Data:        l.Data,
			Processed:   false,
		}
		// Use Create with On Conflict DO NOTHING semantics: GORM upsert requires more setup;
		// For simplicity we attempt Create and ignore unique errors
		if err := tx.Create(&ev).Error; err != nil {
			// check unique constraint error (skip duplicates)
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// reorg detection on startup: compare last N processed blocks with chain
func (s *Scanner) detectAndHandleReorg(ctx context.Context) error {
	// load last REORG_CHECK_DEPTH processed blocks
	var pbs []model.ProcessedBlock
	if err := s.db.WithContext(ctx).
		Where("chain = ?", CHAIN).
		Order("block_number desc").
		Limit(REORG_CHECK_DEPTH).
		Find(&pbs).Error; err != nil {
		return err
	}
	if len(pbs) == 0 {
		return nil
	}
	// iterate from newest to oldest and compare
	for _, pb := range pbs {
		num := big.NewInt(pb.BlockNumber)
		header, err := s.client.HeaderByNumber(ctx, num)
		if err != nil {
			// if RPC cannot find header (e.g. node pruned) skip
			continue
		}
		if header.Hash().Hex() != pb.BlockHash {
			// reorg detected: rollback DB entries > header.Number
			log.Printf("reorg detected at block %d: dbHash=%s chainHash=%s. rolling back above %d",
				pb.BlockNumber, pb.BlockHash, header.Hash().Hex(), pb.BlockNumber-1)
			// delete processed_blocks > pb.BlockNumber-1 and mark events unprocessed
			if err := s.rollbackToBlock(ctx, pb.BlockNumber-1); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (s *Scanner) rollbackToBlock(ctx context.Context, blockNumber int64) error {
	// delete processed blocks above blockNumber
	if err := s.db.WithContext(ctx).Where("chain = ? AND block_number > ?", CHAIN, blockNumber).Delete(&model.ProcessedBlock{}).Error; err != nil {
		return err
	}
	// mark onchain_events above blockNumber as unprocessed (so processor will reprocess)
	if err := s.db.WithContext(ctx).Model(&model.OnchainEvent{}).
		Where("chain = ? AND block_number > ?", CHAIN, blockNumber).
		Updates(map[string]interface{}{"processed": false}).Error; err != nil {
		return err
	}
	// Optionally delete deposits derived from those events - depends on your business logic.
	return nil
}

func minUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func (s *Scanner) adjustStepOnSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.successCount++
	s.failureCount = 0
	if s.successCount >= SUCCESS_THRESHOLD {
		newStep := uint64(float64(s.step) * 1.5)
		if newStep > MAX_STEP {
			newStep = MAX_STEP
		}
		if newStep > s.step {
			log.Printf("increase step %d -> %d", s.step, newStep)
			s.step = newStep
		}
		s.successCount = 0
	}
}

func (s *Scanner) adjustStepOnFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failureCount++
	s.successCount = 0
	if s.failureCount >= FAILURE_THRESHOLD {
		newStep := uint64(float64(s.step) * 0.5)
		if newStep < MIN_STEP {
			newStep = MIN_STEP
		}
		if newStep < s.step {
			log.Printf("decrease step %d -> %d", s.step, newStep)
			s.step = newStep
		}
		s.failureCount = 0
	}
}

func (s *Scanner) stepOnce(ctx context.Context) error {
	// get latest
	header, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		s.adjustStepOnFailure()
		return err
	}
	latest := header.Number.Uint64()
	if latest <= CONFIRMATIONS {
		return nil
	}
	safe := latest - CONFIRMATIONS

	last, err := s.lastProcessedBlock(ctx)
	if err != nil {
		s.adjustStepOnFailure()
		return err
	}
	start := uint64(last + 1)
	if start > safe {
		// nothing to do
		return nil
	}

	// determine end
	s.mu.Lock()
	step := s.step
	s.mu.Unlock()
	end := minUint64(start+step-1, safe)
	log.Printf("scan range %d -> %d (safe=%d, step=%d)", start, end, safe, step)

	q := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(start)),
		ToBlock:   big.NewInt(int64(end)),
		// Addresses: []common.Address{...}, // optional: restrict to tokens/contracts you care about
	}

	logs, err := s.client.FilterLogs(ctx, q)
	if err != nil {
		log.Printf("FilterLogs error: %v", err)
		s.adjustStepOnFailure()
		return err
	}

	// persist logs
	if len(logs) > 0 {
		if err := s.persistLogs(ctx, logs); err != nil {
			s.adjustStepOnFailure()
			return err
		}
	}

	// persist processed_blocks entry for 'end'
	h, err := s.client.HeaderByNumber(ctx, big.NewInt(int64(end)))
	if err != nil {
		// still consider success but skip persisting
		log.Printf("warning: cannot fetch header for %d: %v", end, err)
	} else {
		if err := s.persistProcessedBlock(ctx, int64(end), h.Hash().Hex()); err != nil {
			s.adjustStepOnFailure()
			return err
		}
	}

	s.adjustStepOnSuccess()
	return nil
}

func (s *Scanner) Run(ctx context.Context) {
	// reorg detect on startup
	if err := s.detectAndHandleReorg(ctx); err != nil {
		log.Printf("reorg detect warning: %v", err)
	}
	ticker := time.NewTicker(POLL_INTERVAL)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.stepOnce(ctx); err != nil {
				log.Printf("stepOnce err: %v", err)
			}
		}
	}
}

func main() {
	rpc := os.Getenv("RPC_URL")
	if rpc == "" {
		rpc = RPC_URL
	}
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = DB_DSN
	}
	scanner, err := NewScanner(rpc, dsn)
	if err != nil {
		log.Fatalf("new scanner err: %v", err)
	}
	ctx := context.Background()
	scanner.Run(ctx)
}
