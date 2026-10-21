package domain

import "time"

type ExpenseFilter struct {
	UserID   *int64
	Category Category
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}
