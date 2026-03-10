package domain

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrAlreadyExists       = errors.New("already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrForbidden           = errors.New("forbidden")
	ErrInvalidInput        = errors.New("invalid input")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrEmptyCart           = errors.New("cart is empty")
	ErrOutOfStock          = errors.New("out of stock")
	ErrTooSimilarPasswords = errors.New("passwords are too similar")
	ErrPasswordDontMatch   = errors.New("passwords don't match")
	ErrExpiredToken        = errors.New("expired token")
	ErrInvalidToken        = errors.New("invalid token")
	ErrSessionIsBlocked    = errors.New("session is blocked")
)
