package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/naghinezhad/sms-gateway-system/internal/model"
)

var ErrCustomerNotFound = errors.New("customer not found")
var ErrCustomerExists = errors.New("customer already exists")

type CustomerRepository struct {
	db DBTX
}

func NewCustomerRepository(db DBTX) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) Create(ctx context.Context, customer *model.Customer) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO customers (customer_id, name, created_at)
		VALUES ($1, $2, $3)
	`, customer.CustomerID, customer.Name, customer.CreatedAt)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrCustomerExists
	}

	return err
}

func (r *CustomerRepository) GetByID(ctx context.Context, customerID string) (*model.Customer, error) {
	var customer model.Customer

	err := r.db.QueryRow(ctx, `
		SELECT customer_id, name, created_at
		FROM customers
		WHERE customer_id = $1
	`, customerID).Scan(
		&customer.CustomerID,
		&customer.Name,
		&customer.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}

	return &customer, nil
}
