package service

import (
	"context"
	"crypto/ecdsa"
	_ "encoding/hex"
	"fmt"
	"log"
	"math/big"
	"time"

	_ "github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ==========================
// 数据库模型
// ==========================
type Withdrawal struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"index"`
	Address   string `gorm:"size:64"`
	Amount    string
	Status    string `gorm:"size:20"` // pending / success / failed
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ==========================
// 签名服务
// ==========================
type SignService struct {
	privateKey *ecdsa.PrivateKey
	client     *ethclient.Client
}

func NewSignService(rpcURL string, privateKeyHex string) (*SignService, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, err
	}

	return &SignService{privateKey: privateKey, client: client}, nil
}

func (s *SignService) SignAndSendTx(to common.Address, amount *big.Int) (string, error) {
	fromAddress := crypto.PubkeyToAddress(s.privateKey.PublicKey)

	nonce, err := s.client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", err
	}

	gasPrice, err := s.client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}

	// 构造交易
	tx := types.NewTransaction(nonce, to, amount, uint64(21000), gasPrice, nil)

	chainID, err := s.client.NetworkID(context.Background())
	if err != nil {
		return "", err
	}

	// 签名
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), s.privateKey)
	if err != nil {
		return "", err
	}

	// 广播交易
	err = s.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", err
	}

	return signedTx.Hash().Hex(), nil
}

// ==========================
// 提现服务
// ==========================
type WithdrawalService struct {
	db          *gorm.DB
	signService *SignService
}

func NewWithdrawalService(db *gorm.DB, signService *SignService) *WithdrawalService {
	return &WithdrawalService{db: db, signService: signService}
}

// 提现处理
func (w *WithdrawalService) ProcessWithdrawal(userID uint, to string, amount *big.Int) error {
	// === Step 1: 校验地址合法性 ===
	if !common.IsHexAddress(to) {
		return fmt.Errorf("无效的提现地址: %s", to)
	}

	// === Step 2: 校验余额 (这里简化为假设通过) ===
	// 实际应该从账本表 / redis 中扣减余额
	// 如果余额不足，直接 return error

	// === Step 3: 风控检查 (简化版，只是黑名单校验) ===
	blacklist := []string{"0x1111111111111111111111111111111111111111"}
	for _, bad := range blacklist {
		if to == bad {
			return fmt.Errorf("提现地址在黑名单: %s", to)
		}
	}

	// === Step 4: 创建提现记录 ===
	withdrawal := Withdrawal{
		UserID:  userID,
		Address: to,
		Amount:  amount.String(),
		Status:  "pending",
	}
	if err := w.db.Create(&withdrawal).Error; err != nil {
		return err
	}

	// === Step 5: 调用签名服务广播交易 ===
	txHash, err := w.signService.SignAndSendTx(common.HexToAddress(to), amount)
	if err != nil {
		w.db.Model(&withdrawal).Update("Status", "failed")
		return fmt.Errorf("交易发送失败: %v", err)
	}

	// === Step 6: 更新提现记录 ===
	w.db.Model(&withdrawal).Updates(map[string]interface{}{
		"Status": "success",
	})

	fmt.Printf("提现成功，用户ID=%d，交易哈希=%s\n", userID, txHash)
	return nil
}

// ==========================
// Main 示例
// ==========================
func main() {
	// === 数据库连接 ===
	dsn := "host=localhost port=5432 user=postgres password=123456 dbname=wallet sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 自动建表
	if err := db.AutoMigrate(&Withdrawal{}); err != nil {
		log.Fatal("迁移失败:", err)
	}

	// === 初始化服务 ===
	signService, err := NewSignService("https://mainnet.infura.io/v3/YOUR_PROJECT_ID", "你的私钥HEX")
	if err != nil {
		log.Fatal("初始化签名服务失败:", err)
	}
	withdrawService := NewWithdrawalService(db, signService)

	// === 模拟提现 ===
	amount := big.NewInt(10000000000000000) // 0.01 ETH
	err = withdrawService.ProcessWithdrawal(1, "0x742d35Cc6634C0532925a3b844Bc454e4438f44e", amount)
	if err != nil {
		log.Fatal("提现失败:", err)
	}
}
