package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrWalletNotFound = errors.New("wallet not found")

type WalletRepository struct {
	db DBTX
}

func NewWalletRepository(db DBTX) *WalletRepository {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) Create(ctx context.Context, customerID string, balance int64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO wallets (customer_id, balance, updated_at)
		VALUES ($1, $2, $3)
	`, customerID, balance, time.Now().UTC())
	return err
}

func (r *WalletRepository) GetForUpdate(ctx context.Context, customerID string) (int64, error) {
	var balance int64

	err := r.db.QueryRow(ctx, `
		SELECT balance
		FROM wallets
		WHERE customer_id = $1
		FOR UPDATE
	`, customerID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrWalletNotFound
		}
		return 0, err
	}

	return balance, nil
}

func (r *WalletRepository) GetBalance(ctx context.Context, customerID string) (int64, error) {
	var balance int64

	err := r.db.QueryRow(ctx, `
		SELECT balance
		FROM wallets
		WHERE customer_id = $1
	`, customerID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrWalletNotFound
		}
		return 0, err
	}

	return balance, nil
}

func (r *WalletRepository) UpdateBalance(ctx context.Context, customerID string, balance int64) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE wallets
		SET balance = $1, updated_at = $2
		WHERE customer_id = $3
	`, balance, time.Now().UTC(), customerID)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return ErrWalletNotFound
	}

	return nil
}

func (r *WalletRepository) AddBalance(ctx context.Context, customerID string, amount int64) (int64, error) {
	var balance int64

	err := r.db.QueryRow(ctx, `
		UPDATE wallets
		SET balance = balance + $1, updated_at = $2
		WHERE customer_id = $3
		RETURNING balance
	`, amount, time.Now().UTC(), customerID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrWalletNotFound
		}
		return 0, err
	}

	return balance, nil
}
