package model

import "time"

type Wallet struct {
	CustomerID string    `json:"customerId"`
	Balance    int64     `json:"balance"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type TopUpInput struct {
	Amount int64 `json:"amount" binding:"required"`
}
