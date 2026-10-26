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
	ErrCodeAlreadyExists   = errors.New("code already exists")
	ErrEmailAlreadyReset   = errors.New("email already reset")

	// Email reset is limited to the first hours after registration: later the
	// address can only be changed by the support team.
	ErrEmailResetWindowExpired = errors.New("email reset window has expired, contact support")
	ErrEmailResetNotAllowed    = errors.New("email reset is not allowed for this account, contact support")

	ErrInvalidCode         = errors.New("invalid code")
	ErrTooManyCodeAttempts = errors.New("too many code attempts, request a new code")
)
