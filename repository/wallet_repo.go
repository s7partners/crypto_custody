package repository

import (
	"context"
	"github.com/crypto_custody/model"
	"gorm.io/gorm"
)

type AddressRepository struct {
	db *gorm.DB
}

func NewAddressRepository(db *gorm.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) FindByUserAndCurrency(userId uint64, currency string) (*model.WalletAddress, error) {
	var addr model.WalletAddress
	if err := r.db.Where("user_id=? AND currency=?", userId, currency).First(&addr).Error; err != nil {
		return nil, err
	}
	return &addr, nil
}

type DepositRepository struct {
	db *gorm.DB
}

func NewDepositRepository(db *gorm.DB) *DepositRepository {
	return &DepositRepository{db: db}
}

func (r *DepositRepository) ListDeposits(ctx context.Context, userId uint64, currency string, page, size int) ([]*model.WalletTransaction, int64, error) {
	var list []*model.WalletTransaction
	var total int64
	offset := (page - 1) * size
	r.db.Model(&model.WalletTransaction{}).Where("user_id=? AND currency=? AND type=1", userId, currency).Count(&total)
	if err := r.db.Where("user_id=? AND currency=? AND type=1", userId, currency).Offset(offset).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *DepositRepository) SumAvailableByUserAndCurrency(ctx context.Context, userId uint64, currency string) (float64, error) {
	var sum float64
	err := r.db.Model(&model.WalletTransaction{}).Select("SUM(amount)").Where("user_id=? AND currency=? AND status=2 AND type=1", userId, currency).Scan(&sum).Error
	return sum, err
}

type WithdrawRepository struct {
	db *gorm.DB
}

func NewWithdrawRepository(db *gorm.DB) *WithdrawRepository {
	return &WithdrawRepository{db: db}
}

func (r *WithdrawRepository) Create(withdraw *model.WalletWithdraw) error {
	return r.db.Create(withdraw).Error
}

func (r *WithdrawRepository) ListByUserAndCurrency(ctx context.Context, userId uint64, currency string, page, size int) ([]*model.WalletWithdraw, int64, error) {
	var list []*model.WalletWithdraw
	var total int64
	offset := (page - 1) * size
	r.db.Model(&model.WalletWithdraw{}).Where("user_id=? AND currency=?", userId, currency).Count(&total)
	if err := r.db.Where("user_id=? AND currency=?", userId, currency).Offset(offset).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *WithdrawRepository) SumPendingByUserAndCurrency(ctx context.Context, userId uint64, currency string) (float64, error) {
	var sum float64
	err := r.db.Model(&model.WalletWithdraw{}).Select("SUM(amount)").Where("user_id=? AND currency=? AND status=0", userId, currency).Scan(&sum).Error
	return sum, err
}

type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}
