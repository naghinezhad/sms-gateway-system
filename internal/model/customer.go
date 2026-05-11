package model

import "time"

type Customer struct {
	CustomerID string    `json:"customerId"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"createdAt"`
}

type CreateCustomerInput struct {
	CustomerID string `json:"customerId" binding:"required"`
	Name       string `json:"name" binding:"required"`
}
