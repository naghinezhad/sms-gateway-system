package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naghinezhad/sms-gateway-system/internal/model"
	"github.com/naghinezhad/sms-gateway-system/internal/repository"
)

type CustomerService struct {
	db        *pgxpool.Pool
	customers *repository.CustomerRepository
	wallets   *repository.WalletRepository
}

func NewCustomerService(db *pgxpool.Pool) *CustomerService {
	return &CustomerService{
		db:        db,
		customers: repository.NewCustomerRepository(db),
		wallets:   repository.NewWalletRepository(db),
	}
}

func (s *CustomerService) CreateCustomer(ctx context.Context, input *model.CreateCustomerInput) (*model.Customer, error) {
	if input == nil || input.CustomerID == "" || input.Name == "" {
		return nil, ErrInvalidMessage
	}

	customer := &model.Customer{
		CustomerID: input.CustomerID,
		Name:       input.Name,
		CreatedAt:  time.Now().UTC(),
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	customerRepo := repository.NewCustomerRepository(tx)
	walletRepo := repository.NewWalletRepository(tx)

	if err := customerRepo.Create(ctx, customer); err != nil {
		if err == repository.ErrCustomerExists {
			return nil, ErrCustomerExists
		}
		return nil, err
	}

	if err := walletRepo.Create(ctx, customer.CustomerID, 0); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return customer, nil
}
