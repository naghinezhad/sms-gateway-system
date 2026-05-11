package service

import "errors"

var ErrCustomerNotFound = errors.New("customer not found")
var ErrCustomerExists = errors.New("customer already exists")
var ErrWalletNotFound = errors.New("wallet not found")
var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrInvalidAmount = errors.New("invalid amount")
var ErrInvalidMessage = errors.New("invalid message")
var ErrMessageTooLong = errors.New("message too long")
var ErrMessageNotFound = errors.New("message not found")
var ErrDuplicateEvent = errors.New("duplicate event")
var ErrLockNotAcquired = errors.New("lock not acquired")
