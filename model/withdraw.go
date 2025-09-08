package model

import (
	"time"
)

// 用户表：记录余额（示例中只考虑 ETH，生产要支持多币种）
type User struct {
	ID        uint   `gorm:"primaryKey"`
	Balance   string // 用户余额（wei，decimal string 存储）
	Frozen    string // 冻结金额（未完成的提现）
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 提现表：记录用户发起的提现请求
type Withdrawal struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   // 用户ID
	Chain     string // 区块链，如 "ethereum"
	ToAddress string // 提现地址
	Amount    string // 提现金额（wei，decimal string）
	Fee       string // 手续费（wei，decimal string）
	Status    string // 状态：pending / approved / signed / broadcasted / confirmed / failed
	TxHash    *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 签名请求表：保存待签名的原始交易、签名结果、状态
type SignRequest struct {
	ID           uint   `gorm:"primaryKey"`
	WithdrawalID uint   // 关联的提现记录
	Unsigned     []byte `gorm:"type:bytea"` // 未签名交易 JSON
	Signed       []byte `gorm:"type:bytea"` // 签名结果 RLP
	Status       string // 状态：created / signed / failed
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Nonce 管理表：为每个链上地址维护当前 nonce，避免并发冲突
type ChainNonce struct {
	Chain     string `gorm:"primaryKey"`
	Address   string `gorm:"primaryKey"`
	NextNonce uint64
	UpdatedAt time.Time
}
