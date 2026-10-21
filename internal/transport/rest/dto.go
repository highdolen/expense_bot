package rest

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Money int64

// UnmarshalJSON переводит сумму из рублей в копейки без float64.

func (m *Money) UnmarshalJSON(data []byte) error {
	value := strings.TrimSpace(string(data))

	if value == "" || value == "null" {
		return fmt.Errorf("amount is required")
	}

	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return fmt.Errorf("amount must be positive")
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return fmt.Errorf("invalid amount")
	}

	for _, digit := range parts[0] {
		if digit < '0' || digit > '9' {
			return fmt.Errorf("invalid amount")
		}
	}

	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) == 0 || len(fraction) > 2 {
			return fmt.Errorf("amount must have at most two decimal places")
		}

		for _, digit := range fraction {
			if digit < '0' || digit > '9' {
				return fmt.Errorf("invalid amount")
			}
		}
	}

	if len(fraction) == 1 {
		fraction += "0"
	}
	if len(fraction) == 0 {
		fraction = "00"
	}

	rubles, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || rubles > (int64(^uint64(0)>>1)-99)/100 {
		return fmt.Errorf("amount is too large")
	}

	kopecks, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid amount")
	}

	amount := rubles*100 + kopecks
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	*m = Money(amount)
	return nil
}

func (m Money) MarshalJSON() ([]byte, error) {
	if m < 0 {
		return nil, fmt.Errorf("amount cannot be negative")
	}

	rubles := int64(m) / 100
	kopecks := int64(m) % 100

	return []byte(fmt.Sprintf("%d.%02d", rubles, kopecks)), nil
}

type ExpenseResponse struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Amount    Money     `json:"amount"`
	Category  string    `json:"category"`
	SpentAt   string    `json:"spent_at"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateExpenseRequest struct {
	UserID   int64  `json:"user_id"`
	Amount   Money  `json:"amount"`
	Category string `json:"category"`
	SpentAt  string `json:"spent_at"`
	Comment  string `json:"comment"`
}

type UpdateExpenseRequest struct {
	Amount   *Money  `json:"amount"`
	Category *string `json:"category"`
	SpentAt  *string `json:"spent_at"`
	Comment  *string `json:"comment"`
}

type UserResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	Timezone  string    `json:"timezone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateTimezoneRequest struct {
	Timezone string `json:"timezone"`
}
