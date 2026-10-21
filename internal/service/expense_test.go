package service

import (
	"context"
	"errors"
	"testing"

	"expense_tgbot/internal/domain"
)

type fakeExpenseRepository struct {
	expense  domain.Expense
	expenses []domain.Expense
	total    int64
	err      error

	createdExpense domain.Expense
	updatedExpense domain.Expense
	lastUserID     *int64
	lastID         int64
	lastFilter     domain.ExpenseFilter
	createCalled   bool
	updateCalled   bool
	deleteCalled   bool
}

func (f *fakeExpenseRepository) Create(
	_ context.Context,
	expense domain.Expense,
) (domain.Expense, error) {
	f.createCalled = true
	f.createdExpense = expense

	if f.err != nil {
		return domain.Expense{}, f.err
	}

	return expense, nil
}

func (f *fakeExpenseRepository) GetByID(
	_ context.Context,
	id int64,
	userID *int64,
) (domain.Expense, error) {
	f.lastID = id
	f.lastUserID = userID

	if f.err != nil {
		return domain.Expense{}, f.err
	}

	return f.expense, nil
}

func (f *fakeExpenseRepository) List(
	_ context.Context,
	filter domain.ExpenseFilter,
) ([]domain.Expense, error) {
	f.lastFilter = filter

	if f.err != nil {
		return nil, f.err
	}

	return f.expenses, nil
}

func (f *fakeExpenseRepository) Total(
	_ context.Context,
	filter domain.ExpenseFilter,
) (int64, error) {
	f.lastFilter = filter

	if f.err != nil {
		return 0, f.err
	}

	return f.total, nil
}

func (f *fakeExpenseRepository) Update(
	_ context.Context,
	expense domain.Expense,
	userID *int64,
) (domain.Expense, error) {
	f.updateCalled = true
	f.updatedExpense = expense
	f.lastUserID = userID

	if f.err != nil {
		return domain.Expense{}, f.err
	}

	return expense, nil
}

func (f *fakeExpenseRepository) Delete(
	_ context.Context,
	id int64,
	userID *int64,
) error {
	f.deleteCalled = true
	f.lastID = id
	f.lastUserID = userID

	return f.err
}

func TestExpenseServiceCreate(t *testing.T) {
	repo := &fakeExpenseRepository{}
	service := NewExpenseService(repo)

	expense := domain.Expense{
		UserID:   999, // Сервис должен заменить владельца.
		Amount:   35000,
		Category: domain.CategoryFood,
	}

	got, err := service.Create(context.Background(), 1, expense)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.UserID != 1 {
		t.Fatalf("got user ID %d, want 1", got.UserID)
	}

	if got.Amount != 35000 {
		t.Fatalf("got amount %d, want 35000", got.Amount)
	}

	if !repo.createCalled {
		t.Fatal("repository Create was not called")
	}

	if repo.createdExpense.UserID != 1 {
		t.Fatalf("repository received user ID %d, want 1", repo.createdExpense.UserID)
	}
}

func TestExpenseServiceCreateRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		expense domain.Expense
	}{
		{
			name:   "invalid user ID",
			userID: 0,
			expense: domain.Expense{
				Amount:   10000,
				Category: domain.CategoryFood,
			},
		},
		{
			name:   "invalid amount",
			userID: 1,
			expense: domain.Expense{
				Amount:   0,
				Category: domain.CategoryFood,
			},
		},
		{
			name:   "invalid category",
			userID: 1,
			expense: domain.Expense{
				Amount:   10000,
				Category: domain.Category("unknown"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeExpenseRepository{}
			service := NewExpenseService(repo)

			_, err := service.Create(context.Background(), tt.userID, tt.expense)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}

			if repo.createCalled {
				t.Fatal("repository Create should not be called")
			}
		})
	}
}

func TestExpenseServiceGetByID(t *testing.T) {
	repo := &fakeExpenseRepository{
		expense: domain.Expense{
			ID:       5,
			UserID:   2,
			Amount:   25000,
			Category: domain.CategoryFood,
		},
	}
	service := NewExpenseService(repo)

	userID := int64(2)
	got, err := service.GetByID(context.Background(), 5, &userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ID != 5 {
		t.Fatalf("got ID %d, want 5", got.ID)
	}

	if repo.lastID != 5 {
		t.Fatalf("repository received ID %d, want 5", repo.lastID)
	}
}

func TestExpenseServiceGetByIDRejectsInvalidID(t *testing.T) {
	repo := &fakeExpenseRepository{}
	service := NewExpenseService(repo)

	_, err := service.GetByID(context.Background(), 0, nil)
	if err == nil {
		t.Fatal("expected an error for invalid expense ID")
	}

	if repo.lastID != 0 {
		t.Fatal("repository should not be called")
	}
}

func TestExpenseServiceList(t *testing.T) {
	repo := &fakeExpenseRepository{
		expenses: []domain.Expense{
			{ID: 1, Amount: 10000, Category: domain.CategoryFood},
			{ID: 2, Amount: 20000, Category: domain.CategoryTransport},
		},
	}
	service := NewExpenseService(repo)

	got, err := service.List(context.Background(), domain.ExpenseFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d expenses, want 2", len(got))
	}
}

func TestExpenseServiceListRejectsInvalidUserID(t *testing.T) {
	repo := &fakeExpenseRepository{}
	service := NewExpenseService(repo)

	userID := int64(0)
	_, err := service.List(context.Background(), domain.ExpenseFilter{
		UserID: &userID,
	})

	if err == nil {
		t.Fatal("expected an error for invalid user ID")
	}

	if repo.lastFilter.UserID != nil {
		t.Fatal("repository should not be called")
	}
}

func TestExpenseServiceTotal(t *testing.T) {
	repo := &fakeExpenseRepository{total: 45000}
	service := NewExpenseService(repo)

	got, err := service.Total(context.Background(), domain.ExpenseFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != 45000 {
		t.Fatalf("got total %d, want 45000", got)
	}
}

func TestExpenseServiceUpdate(t *testing.T) {
	repo := &fakeExpenseRepository{}
	service := NewExpenseService(repo)

	expense := domain.Expense{
		ID:       3,
		UserID:   1,
		Amount:   15000,
		Category: domain.CategoryFood,
	}

	userID := int64(1)
	got, err := service.Update(context.Background(), expense, &userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ID != 3 {
		t.Fatalf("got ID %d, want 3", got.ID)
	}

	if !repo.updateCalled {
		t.Fatal("repository Update was not called")
	}
}

func TestExpenseServiceUpdateRejectsInvalidAmount(t *testing.T) {
	repo := &fakeExpenseRepository{}
	service := NewExpenseService(repo)

	_, err := service.Update(context.Background(), domain.Expense{
		ID:       3,
		Amount:   0,
		Category: domain.CategoryFood,
	}, nil)

	if err == nil {
		t.Fatal("expected an error for invalid amount")
	}

	if repo.updateCalled {
		t.Fatal("repository Update should not be called")
	}
}

func TestExpenseServiceDelete(t *testing.T) {
	repo := &fakeExpenseRepository{}
	service := NewExpenseService(repo)

	userID := int64(2)
	err := service.Delete(context.Background(), 7, &userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.deleteCalled {
		t.Fatal("repository Delete was not called")
	}

	if repo.lastID != 7 {
		t.Fatalf("repository received ID %d, want 7", repo.lastID)
	}
}

func TestExpenseServiceDeleteRejectsInvalidID(t *testing.T) {
	repo := &fakeExpenseRepository{}
	service := NewExpenseService(repo)

	err := service.Delete(context.Background(), 0, nil)
	if err == nil {
		t.Fatal("expected an error for invalid expense ID")
	}

	if repo.deleteCalled {
		t.Fatal("repository Delete should not be called")
	}
}

func TestExpenseServiceReturnsRepositoryError(t *testing.T) {
	repoErr := errors.New("database unavailable")
	repo := &fakeExpenseRepository{err: repoErr}
	service := NewExpenseService(repo)

	_, err := service.Create(context.Background(), 1, domain.Expense{
		Amount:   10000,
		Category: domain.CategoryFood,
	})

	if !errors.Is(err, repoErr) {
		t.Fatalf("got error %v, want %v", err, repoErr)
	}
}
