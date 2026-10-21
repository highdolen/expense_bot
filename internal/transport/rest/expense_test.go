package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expense_tgbot/internal/domain"
)

type fakeExpenseManager struct {
	expenses []domain.Expense
	expense  domain.Expense
	total    int64
	err      error

	lastFilter domain.ExpenseFilter
	lastID     int64
	lastUpdate domain.Expense
}

func (f *fakeExpenseManager) Create(
	_ context.Context,
	userID int64,
	expense domain.Expense,
) (domain.Expense, error) {
	if f.err != nil {
		return domain.Expense{}, f.err
	}

	expense.ID = 1
	expense.UserID = userID
	f.expense = expense
	return expense, nil
}

func (f *fakeExpenseManager) GetByID(
	_ context.Context,
	id int64,
	_ *int64,
) (domain.Expense, error) {
	f.lastID = id
	if f.err != nil {
		return domain.Expense{}, f.err
	}
	return f.expense, nil
}

func (f *fakeExpenseManager) List(
	_ context.Context,
	filter domain.ExpenseFilter,
) ([]domain.Expense, error) {
	f.lastFilter = filter
	if f.err != nil {
		return nil, f.err
	}
	return f.expenses, nil
}

func (f *fakeExpenseManager) Total(
	_ context.Context,
	filter domain.ExpenseFilter,
) (int64, error) {
	f.lastFilter = filter
	if f.err != nil {
		return 0, f.err
	}
	return f.total, nil
}

func (f *fakeExpenseManager) Update(
	_ context.Context,
	expense domain.Expense,
	_ *int64,
) (domain.Expense, error) {
	f.lastUpdate = expense
	if f.err != nil {
		return domain.Expense{}, f.err
	}
	return expense, nil
}

func (f *fakeExpenseManager) Delete(
	_ context.Context,
	id int64,
	_ *int64,
) error {
	f.lastID = id
	return f.err
}

func TestExpenseCreate(t *testing.T) {
	manager := &fakeExpenseManager{}
	handler := NewExpenseHandler(manager)

	body := `{
		"user_id": 7,
		"amount": 350.50,
		"category": "food",
		"spent_at": "2026-10-20",
		"comment": "  coffee  "
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/expenses", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("got status %d, want %d; body: %s",
			rec.Code, http.StatusCreated, rec.Body.String())
	}

	if manager.expense.UserID != 7 {
		t.Errorf("got user ID %d, want 7", manager.expense.UserID)
	}
	if manager.expense.Amount != 35050 {
		t.Errorf("got amount %d kopecks, want 35050", manager.expense.Amount)
	}
	if manager.expense.Category != domain.CategoryFood {
		t.Errorf("got category %q, want %q", manager.expense.Category, domain.CategoryFood)
	}
	if manager.expense.Comment != "coffee" {
		t.Errorf("got comment %q, want %q", manager.expense.Comment, "coffee")
	}

	var response ExpenseResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if response.ID != 1 {
		t.Errorf("got response ID %d, want 1", response.ID)
	}
}

func TestExpenseCreateRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "invalid JSON",
			body: `{"user_id":`,
		},
		{
			name: "invalid user ID",
			body: `{"user_id":0,"amount":100,"category":"food","spent_at":"2026-10-20"}`,
		},
		{
			name: "zero amount",
			body: `{"user_id":1,"amount":0,"category":"food","spent_at":"2026-10-20"}`,
		},
		{
			name: "unknown category",
			body: `{"user_id":1,"amount":100,"category":"unknown","spent_at":"2026-10-20"}`,
		},
		{
			name: "invalid date",
			body: `{"user_id":1,"amount":100,"category":"food","spent_at":"20-10-2026"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewExpenseHandler(&fakeExpenseManager{})
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/expenses",
				strings.NewReader(tt.body),
			)
			rec := httptest.NewRecorder()

			handler.Create(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("got status %d, want %d; body: %s",
					rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}

func TestExpenseList(t *testing.T) {
	spentAt := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	manager := &fakeExpenseManager{
		expenses: []domain.Expense{
			{
				ID:       1,
				UserID:   7,
				Amount:   35000,
				Category: domain.CategoryFood,
				SpentAt:  spentAt,
				Comment:  "coffee",
			},
		},
		total: 35000,
	}
	handler := NewExpenseHandler(manager)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/expenses?user_id=7&category=food&from=2026-10-01&to=2026-10-31&limit=10&offset=5",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d; body: %s",
			rec.Code, http.StatusOK, rec.Body.String())
	}

	if manager.lastFilter.UserID == nil || *manager.lastFilter.UserID != 7 {
		t.Fatal("expected filter user ID to be 7")
	}
	if manager.lastFilter.Category != domain.CategoryFood {
		t.Errorf("got category %q, want food", manager.lastFilter.Category)
	}
	if manager.lastFilter.Limit != 10 || manager.lastFilter.Offset != 5 {
		t.Errorf("got limit=%d offset=%d, want limit=10 offset=5",
			manager.lastFilter.Limit, manager.lastFilter.Offset)
	}

	var response struct {
		Items []ExpenseResponse `json:"items"`
		Total Money             `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(response.Items))
	}
	if response.Total != 35000 {
		t.Errorf("got total %d, want 35000", response.Total)
	}
}

func TestExpenseUpdate(t *testing.T) {
	manager := &fakeExpenseManager{
		expense: domain.Expense{
			ID:       3,
			UserID:   7,
			Amount:   10000,
			Category: domain.CategoryFood,
			SpentAt:  time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
			Comment:  "old comment",
		},
	}
	handler := NewExpenseHandler(manager)

	body := `{"amount":125.50,"category":"shopping","comment":"  groceries  "}`
	req := httptest.NewRequest(http.MethodPut, "/api/expenses/3", strings.NewReader(body))
	req.SetPathValue("id", "3")
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d; body: %s",
			rec.Code, http.StatusOK, rec.Body.String())
	}

	if manager.lastID != 3 {
		t.Errorf("got ID %d, want 3", manager.lastID)
	}
	if manager.lastUpdate.Amount != 12550 {
		t.Errorf("got amount %d, want 12550", manager.lastUpdate.Amount)
	}
	if manager.lastUpdate.Category != domain.CategoryShopping {
		t.Errorf("got category %q, want shopping", manager.lastUpdate.Category)
	}
	if manager.lastUpdate.Comment != "groceries" {
		t.Errorf("got comment %q, want groceries", manager.lastUpdate.Comment)
	}
}

func TestExpenseDelete(t *testing.T) {
	manager := &fakeExpenseManager{}
	handler := NewExpenseHandler(manager)

	req := httptest.NewRequest(http.MethodDelete, "/api/expenses/5", nil)
	req.SetPathValue("id", "5")
	rec := httptest.NewRecorder()

	handler.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusNoContent)
	}
	if manager.lastID != 5 {
		t.Errorf("got deleted ID %d, want 5", manager.lastID)
	}
}

func TestExpenseListServiceError(t *testing.T) {
	manager := &fakeExpenseManager{
		err: errors.New("database unavailable"),
	}
	handler := NewExpenseHandler(manager)

	req := httptest.NewRequest(http.MethodGet, "/api/expenses", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want %d",
			rec.Code, http.StatusInternalServerError)
	}
}
