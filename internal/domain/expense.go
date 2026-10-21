package domain

import "time"

type Expense struct {
	ID        int64
	UserID    int64
	Amount    int64
	Category  Category
	SpentAt   time.Time
	Comment   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
