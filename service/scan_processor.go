package service

import (
	"context"
	"encoding/json"
	"fmt"
	model "github.com/crypto_custody/model"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"math/big"
	"os"
	"strings"
	"time"
)

const (
	PROCESS_DB_DSN        = "host=localhost user=postgres password=postgres dbname=custody sslmode=disable"
	PROCESS_CHAIN         = "ethereum"
	BATCH_PROCESS_SIZE    = 100
	POLL_PROCESS_INTERVAL = 2 * time.Second
)

// Transfer event signature: Transfer(address,address,uint256)
var transferEventSig = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

// minimal ERC20 ABI for Transfer decoding
const erc20ABIJSON = `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`

type Processor struct {
	db  *gorm.DB
	erc abi.ABI
}

func NewProcessor(dsn string) (*Processor, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := model.AutoMigrate(db); err != nil {
		return nil, err
	}
	erc, err := abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		return nil, err
	}
	return &Processor{db: db, erc: erc}, nil
}

func (p *Processor) fetchPendingEvents(ctx context.Context, limit int) ([]model.OnchainEvent, error) {
	var evs []model.OnchainEvent
	if err := p.db.WithContext(ctx).
		Where("chain = ? AND processed = false", PROCESS_CHAIN).
		Order("block_number asc, id asc").
		Limit(limit).
		Find(&evs).Error; err != nil {
		return nil, err
	}
	return evs, nil
}

func (p *Processor) markEventProcessedTx(tx *gorm.DB, evID uint) error {
	return tx.Model(&model.OnchainEvent{}).Where("id = ?", evID).Update("processed", true).Error
}

// parseTransfer tries to decode ERC20 Transfer: returns (from,to,value) or error
func (p *Processor) parseTransfer(ev model.OnchainEvent) (common.Address, common.Address, *big.Int, error) {
	// topic[0] must be transferEventSig
	var topics []common.Hash
	if err := json.Unmarshal([]byte(ev.Topics), &topics); err != nil {
		return common.Address{}, common.Address{}, nil, fmt.Errorf("unmarshal topics: %w", err)
	}
	if len(topics) == 0 {
		return common.Address{}, common.Address{}, nil, fmt.Errorf("no topics")
	}
	if topics[0] != transferEventSig {
		return common.Address{}, common.Address{}, nil, fmt.Errorf("not transfer event")
	}
	// decode indexed: topics[1] = from, topics[2]=to, data = value
	if len(topics) < 3 {
		return common.Address{}, common.Address{}, nil, fmt.Errorf("topics len < 3")
	}
	from := common.BytesToAddress(topics[1].Bytes()[12:])
	to := common.BytesToAddress(topics[2].Bytes()[12:])
	// value in data (non-indexed)
	var out struct{ Value *big.Int }
	err := p.erc.UnpackIntoInterface(&out, "Transfer", ev.Data)
	if err != nil {
		// sometimes event might encode differently; attempt raw parsing
		return common.Address{}, common.Address{}, nil, fmt.Errorf("abi unpack err: %w", err)
	}
	return from, to, out.Value, nil
}

func (p *Processor) processEvent(ctx context.Context, ev model.OnchainEvent) error {
	// start tx
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// optimistic check if already processed
		var exist model.OnchainEvent
		if err := tx.Clauses().Where("id = ? AND processed = false", ev.ID).First(&exist).Error; err != nil {
			// either not found or already processed
			return nil
		}

		// try parse transfer
		_, to, value, err := p.parseTransfer(ev)
		if err != nil {
			// Not an ERC20 Transfer or parse failed; mark processed to skip (or keep unprocessed if you want to handle other types)
			if err := p.markEventProcessedTx(tx, ev.ID); err != nil {
				return err
			}
			return nil
		}

		// check if 'to' is in our address pool
		var ap model.AddressPool
		if err := tx.Where("address = ?", strings.ToLower(to.Hex())).First(&ap).Error; err != nil {
			// try uppercase format or no match: mark processed and skip (or keep unprocessed for manual review)
			_ = p.markEventProcessedTx(tx, ev.ID)
			return nil
		}

		// create deposit if not exist for same tx/logindex
		// check duplicate deposit by tx_hash + to_address + amount
		var dup model.Deposit
		if err := tx.Where("tx_hash = ? AND to_address = ?", ev.TxHash, strings.ToLower(to.Hex())).First(&dup).Error; err == nil {
			// already exists -> mark event processed and return
			if err := p.markEventProcessedTx(tx, ev.ID); err != nil {
				return err
			}
			return nil
		}

		amountText := value.Text(10)
		tokenAddr := ev.Address // contract address

		dep := model.Deposit{
			Chain:       PROCESS_CHAIN,
			Token:       &tokenAddr,
			ToAddress:   strings.ToLower(to.Hex()),
			UserID:      ap.UserID,
			Amount:      amountText,
			TxHash:      ev.TxHash,
			BlockNumber: ev.BlockNumber,
			Confirmed:   true, // since scanner only processes after confirmations
		}
		if err := tx.Create(&dep).Error; err != nil {
			return err
		}

		// mark event processed
		if err := p.markEventProcessedTx(tx, ev.ID); err != nil {
			return err
		}

		// optionally: update address_pool used flag and link to user
		if ap.ID != 0 && !ap.Used {
			ap.Used = true
			ap.UserID = ap.UserID // already set
			if err := tx.Save(&ap).Error; err != nil {
				return err
			}
		}

		// success transaction commit
		return nil
	})
}

func (p *Processor) Run(ctx context.Context) {
	t := time.NewTicker(POLL_PROCESS_INTERVAL)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			evs, err := p.fetchPendingEvents(ctx, BATCH_PROCESS_SIZE)
			if err != nil {
				log.Printf("fetch pending events err: %v", err)
				continue
			}
			if len(evs) == 0 {
				continue
			}
			for _, ev := range evs {
				if err := p.processEvent(ctx, ev); err != nil {
					log.Printf("process event id=%d err: %v", ev.ID, err)
				}
			}
		}
	}
}

func main() {
	dsn := PROCESS_DB_DSN
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		dsn = v
	}
	p, err := NewProcessor(dsn)
	if err != nil {
		log.Fatalf("new processor err: %v", err)
	}
	ctx := context.Background()
	p.Run(ctx)
}
