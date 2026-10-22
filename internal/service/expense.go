package service

import (
	"context"
	"fmt"

	"expense_tgbot/internal/domain"
)

type ExpenseRepository interface {
	Create(ctx context.Context, expense domain.Expense) (domain.Expense, error)
	GetByID(ctx context.Context, id int64, userID *int64) (domain.Expense, error)
	List(ctx context.Context, filter domain.ExpenseFilter) ([]domain.Expense, error)
	Total(ctx context.Context, filter domain.ExpenseFilter) (int64, error)
	Count(ctx context.Context, filter domain.ExpenseFilter) (int64, error)
	Update(ctx context.Context, expense domain.Expense, userID *int64) (domain.Expense, error)
	Delete(ctx context.Context, id int64, userID *int64) error
}

type ExpenseService struct {
	repo ExpenseRepository
}

func NewExpenseService(repo ExpenseRepository) *ExpenseService {
	return &ExpenseService{repo: repo}
}

func (s *ExpenseService) Create(
	ctx context.Context,
	userID int64,
	expense domain.Expense,
) (domain.Expense, error) {
	if userID <= 0 {
		return domain.Expense{}, fmt.Errorf("invalid user ID")
	}

	if err := validateExpense(expense); err != nil {
		return domain.Expense{}, err
	}

	// ID владельца задаётся сервисом, а не передаётся из AI.
	expense.UserID = userID

	return s.repo.Create(ctx, expense)
}

func (s *ExpenseService) GetByID(
	ctx context.Context,
	id int64,
	userID *int64,
) (domain.Expense, error) {
	if id <= 0 {
		return domain.Expense{}, fmt.Errorf("invalid expense ID")
	}

	if userID != nil && *userID <= 0 {
		return domain.Expense{}, fmt.Errorf("invalid user ID")
	}

	return s.repo.GetByID(ctx, id, userID)
}

func (s *ExpenseService) List(
	ctx context.Context,
	filter domain.ExpenseFilter,
) ([]domain.Expense, error) {
	if err := validateExpenseFilter(filter); err != nil {
		return nil, err
	}

	return s.repo.List(ctx, filter)
}

func (s *ExpenseService) Total(
	ctx context.Context,
	filter domain.ExpenseFilter,
) (int64, error) {
	if err := validateExpenseFilter(filter); err != nil {
		return 0, err
	}

	// Репозиторий учитывает фильтры, но игнорирует limit и offset.
	return s.repo.Total(ctx, filter)
}

func (s *ExpenseService) Count(
	ctx context.Context,
	filter domain.ExpenseFilter,
) (int64, error) {
	if err := validateExpenseFilter(filter); err != nil {
		return 0, err
	}

	return s.repo.Count(ctx, filter)
}

func (s *ExpenseService) Update(
	ctx context.Context,
	expense domain.Expense,
	userID *int64,
) (domain.Expense, error) {
	if expense.ID <= 0 {
		return domain.Expense{}, fmt.Errorf("invalid expense ID")
	}

	if userID != nil && *userID <= 0 {
		return domain.Expense{}, fmt.Errorf("invalid user ID")
	}

	if err := validateExpense(expense); err != nil {
		return domain.Expense{}, err
	}

	return s.repo.Update(ctx, expense, userID)
}

func (s *ExpenseService) Delete(
	ctx context.Context,
	id int64,
	userID *int64,
) error {
	if id <= 0 {
		return fmt.Errorf("invalid expense ID")
	}

	if userID != nil && *userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	return s.repo.Delete(ctx, id, userID)
}

func validateExpense(expense domain.Expense) error {
	if expense.Amount <= 0 {
		return domain.ErrInvalidAmount
	}

	if !expense.Category.IsValid() {
		return domain.ErrInvalidCategory
	}

	return nil
}

func validateExpenseFilter(filter domain.ExpenseFilter) error {
	if filter.UserID != nil && *filter.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	return nil
}
