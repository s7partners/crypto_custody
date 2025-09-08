package model

import (
	"time"

	"gorm.io/gorm"
)

type ProcessedBlock struct {
	ID          uint   `gorm:"primaryKey"`
	Chain       string `gorm:"size:32;index:idx_chain_block,unique"`
	BlockNumber int64  `gorm:"index:idx_chain_block,unique"`
	BlockHash   string `gorm:"size:128"`
	CreatedAt   time.Time
}

type OnchainEvent struct {
	ID          uint   `gorm:"primaryKey"`
	Chain       string `gorm:"size:32;index:idx_event_unique,priority:1"`
	BlockNumber int64  `gorm:"index"`
	BlockHash   string `gorm:"size:128"`
	TxHash      string `gorm:"size:128;index:idx_event_unique,priority:2"`
	LogIndex    int    `gorm:"index:idx_event_unique,priority:3"`
	Address     string `gorm:"size:128"`  // contract address that emitted the log
	Topics      string `gorm:"type:text"` // JSON-encoded or concatenated topics for simplicity
	Data        []byte `gorm:"type:bytea"`
	Processed   bool   `gorm:"index"`
	CreatedAt   time.Time
}

type AddressPool struct {
	ID        uint   `gorm:"primaryKey"`
	Chain     string `gorm:"size:32;index"`
	Address   string `gorm:"size:128;uniqueIndex"`
	UserID    *int64
	Used      bool
	CreatedAt time.Time
}

type Deposit struct {
	ID          uint    `gorm:"primaryKey"`
	Chain       string  `gorm:"size:32;index"`
	Token       *string `gorm:"size:128;null"` // token contract address for ERC20, nil for native
	ToAddress   string  `gorm:"size:128;index"`
	UserID      *int64
	Amount      string `gorm:"type:text"` // use string to store big integers (wei/satoshi)
	TxHash      string `gorm:"size:128;index"`
	BlockNumber int64  `gorm:"index"`
	Confirmed   bool
	CreatedAt   time.Time
}

// helper: create tables
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&ProcessedBlock{}, &OnchainEvent{}, &AddressPool{}, &Deposit{})
}
