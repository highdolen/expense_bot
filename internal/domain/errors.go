package domain

import "errors"

var (
	ErrExpenseNotFound = errors.New("expense not found")
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidAmount   = errors.New("invalid expense amount")
	ErrInvalidCategory = errors.New("invalid expense category")
	ErrInvalidTimezone = errors.New("invalid timezone")
)
