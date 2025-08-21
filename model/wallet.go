package model

import (
	"time"
)

// 钱包充值地址表（wallet_address）
type WalletAddress struct {
	ID        uint64    `gorm:"primaryKey;column:id" json:"id"`
	UserID    uint64    `gorm:"column:user_id;not null" json:"user_id"`
	Currency  string    `gorm:"column:currency;type:varchar(16);not null" json:"currency"`
	Address   string    `gorm:"column:address;type:varchar(256);uniqueIndex" json:"address"`
	HDPath    string    `gorm:"column:hd_path;type:varchar(128)" json:"hd_path"`
	Type      int8      `gorm:"column:type;not null;comment:0=热钱包,1=冷钱包" json:"type"`
	Status    int8      `gorm:"column:status;not null;default:0;comment:0=启用,1=停用" json:"status"`
	CreatedAt time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
}

// 钱包充值记录表（wallet_deposit）
type WalletDeposit struct {
	ID        uint64    `gorm:"primaryKey;column:id" json:"id"`
	UserID    uint64    `gorm:"column:user_id;not null" json:"user_id"`
	Currency  string    `gorm:"column:currency;type:varchar(16);not null" json:"currency"`
	RefID     uint64    `gorm:"column:ref_id" json:"ref_id"`
	Type      int8      `gorm:"column:type;not null;comment:1=充值,2=提现,3=手续费" json:"type"`
	Amount    float64   `gorm:"column:amount;type:decimal(32,8);not null" json:"amount"`
	Status    int8      `gorm:"column:status;not null;default:0;comment:0=处理中,1=成功,2=失败" json:"status"`
	CreatedAt time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
}

// 钱包提现记录表（wallet_withdraw）
type WalletWithdraw struct {
	ID        uint64    `gorm:"primaryKey;column:id" json:"id"`
	UserID    uint64    `gorm:"column:user_id;not null" json:"user_id"`
	Currency  string    `gorm:"column:currency;type:varchar(16);not null" json:"currency"`
	Address   string    `gorm:"column:address;type:varchar(256);not null" json:"address"`
	Amount    float64   `gorm:"column:amount;type:decimal(32,8);not null" json:"amount"`
	TxID      string    `gorm:"column:tx_id;type:varchar(128)" json:"tx_id"`
	Status    int8      `gorm:"column:status;not null;default:0;comment:0=Pending,1=Signed,2=Broadcasted,3=Confirmed,4=Failed" json:"status"`
	CreatedAt time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdatedAt time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}

// 钱包资金流水表（wallet_transaction）
type WalletTransaction struct {
	ID            uint64    `gorm:"primaryKey;column:id" json:"id"`
	UserID        uint64    `gorm:"column:user_id;not null" json:"user_id"`
	Currency      string    `gorm:"column:currency;type:varchar(16);not null" json:"currency"`
	Address       string    `gorm:"column:address;type:varchar(256)" json:"address"`
	TxID          string    `gorm:"column:tx_id;type:varchar(128)" json:"tx_id"`
	Amount        float64   `gorm:"column:amount;type:decimal(32,8);not null" json:"amount"`
	Confirmations int       `gorm:"column:confirmations;default:0" json:"confirmations"`
	Status        int8      `gorm:"column:status;not null;default:0;comment:0=Pending,1=Confirmed,2=Credited,3=Failed" json:"status"`
	BlockHeight   uint64    `gorm:"column:block_height" json:"block_height"`
	CreatedAt     time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdatedAt     time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}
