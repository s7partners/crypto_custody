package service

import (
	"context"
	"github.com/crypto_custody/model"
	"github.com/crypto_custody/repository"
)

type WalletService struct {
	addressRepo     *repository.AddressRepository
	depositRepo     *repository.DepositRepository
	withdrawRepo    *repository.WithdrawRepository
	transactionRepo *repository.TransactionRepository
}

func NewWalletService(addr *repository.AddressRepository,
	dep *repository.DepositRepository,
	withd *repository.WithdrawRepository,
	tx *repository.TransactionRepository) *WalletService {
	return &WalletService{
		addressRepo:     addr,
		depositRepo:     dep,
		withdrawRepo:    withd,
		transactionRepo: tx,
	}
}

// 申请充值地址
func (s *WalletService) GetDepositAddress(userID uint64, currency string) (*model.WalletAddress, error) {
	return s.addressRepo.FindByUserAndCurrency(userID, currency)
}

// 提交提现请求
func (s *WalletService) RequestWithdraw(userID uint64, currency, address string, amount float64) (*model.WalletWithdraw, error) {
	withdraw := &model.WalletWithdraw{
		UserID:   userID,
		Currency: currency,
		Address:  address,
		Amount:   amount,
		Status:   0, // 待处理
	}
	if err := s.withdrawRepo.Create(withdraw); err != nil {
		return nil, err
	}
	return withdraw, nil
}

// 查询充值记录
func (s *WalletService) GetDepositHistory(ctx context.Context, userId uint64, currency string, page, size int) ([]*model.WalletTransaction, int64, error) {
	return s.depositRepo.ListDeposits(ctx, userId, currency, page, size)
}

// 查询提现记录
func (s *WalletService) GetWithdrawHistory(ctx context.Context, userId uint64, currency string, page, size int) ([]*model.WalletWithdraw, int64, error) {
	return s.withdrawRepo.ListByUserAndCurrency(ctx, userId, currency, page, size)
}

// 查询账户余额
func (s *WalletService) GetBalance(ctx context.Context, userId uint64, currency string) (available float64, frozen float64, err error) {
	available, err = s.depositRepo.SumAvailableByUserAndCurrency(ctx, userId, currency)
	if err != nil {
		return
	}
	frozen, err = s.withdrawRepo.SumPendingByUserAndCurrency(ctx, userId, currency)
	return
}
