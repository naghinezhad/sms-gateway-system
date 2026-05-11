package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naghinezhad/sms-gateway-system/internal/repository"
)

type WalletService struct {
	customers *repository.CustomerRepository
	wallets   *repository.WalletRepository
}

func NewWalletService(db *pgxpool.Pool) *WalletService {
	return &WalletService{
		customers: repository.NewCustomerRepository(db),
		wallets:   repository.NewWalletRepository(db),
	}
}

func (s *WalletService) TopUp(ctx context.Context, customerID string, amount int64) (int64, error) {
	if amount <= 0 {
		return 0, ErrInvalidAmount
	}

	if _, err := s.customers.GetByID(ctx, customerID); err != nil {
		if err == repository.ErrCustomerNotFound {
			return 0, ErrCustomerNotFound
		}
		return 0, err
	}

	balance, err := s.wallets.AddBalance(ctx, customerID, amount)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return 0, ErrWalletNotFound
		}
		return 0, err
	}

	return balance, nil
}

func (s *WalletService) GetBalance(ctx context.Context, customerID string) (int64, error) {
	if _, err := s.customers.GetByID(ctx, customerID); err != nil {
		if err == repository.ErrCustomerNotFound {
			return 0, ErrCustomerNotFound
		}
		return 0, err
	}

	balance, err := s.wallets.GetBalance(ctx, customerID)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return 0, ErrWalletNotFound
		}
		return 0, err
	}

	return balance, nil
}
