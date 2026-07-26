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
	ErrBuiltToken          = errors.New("error built token")
	ErrSessionIsBlocked    = errors.New("session is blocked")
	ErrDeviceMistake       = errors.New("device mistake")
	ErrRedisNotFound       = errors.New("key does not exists")
	ErrAuthBlock           = errors.New("authentication temporally blocked")
	ErrFileTooLarge        = errors.New("file too large")
	ErrInvalidFileType     = errors.New("invalid file type")
	ErrCategoryNotFound    = errors.New("category not found")
	ErrMoreThan5Claimed    = errors.New("claimed more than 5 products to review")
)
