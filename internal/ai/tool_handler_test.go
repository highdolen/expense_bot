package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"expense_tgbot/internal/domain"
)

type fakeExpenseManager struct {
	created      domain.Expense
	createUserID int64
	listedFilter domain.ExpenseFilter
	listResult   []domain.Expense
	totalResult  int64
	gotExpense   domain.Expense
	getID        int64
	getUserID    *int64
	updated      domain.Expense
	updateUserID *int64
	deletedID    int64
	deleteUserID *int64
	createErr    error
	listErr      error
	totalErr     error
	getErr       error
	updateErr    error
	deleteErr    error
}

func (f *fakeExpenseManager) Create(_ context.Context, userID int64, expense domain.Expense) (domain.Expense, error) {
	f.createUserID, f.created = userID, expense
	if f.createErr != nil {
		return domain.Expense{}, f.createErr
	}
	expense.ID, expense.UserID = 11, userID
	return expense, nil
}
func (f *fakeExpenseManager) List(_ context.Context, filter domain.ExpenseFilter) ([]domain.Expense, error) {
	f.listedFilter = filter
	return f.listResult, f.listErr
}
func (f *fakeExpenseManager) Total(_ context.Context, _ domain.ExpenseFilter) (int64, error) {
	return f.totalResult, f.totalErr
}
func (f *fakeExpenseManager) GetByID(_ context.Context, id int64, userID *int64) (domain.Expense, error) {
	f.getID, f.getUserID = id, userID
	if f.getErr != nil {
		return domain.Expense{}, f.getErr
	}
	return f.gotExpense, nil
}
func (f *fakeExpenseManager) Update(_ context.Context, expense domain.Expense, userID *int64) (domain.Expense, error) {
	f.updated, f.updateUserID = expense, userID
	if f.updateErr != nil {
		return domain.Expense{}, f.updateErr
	}
	return expense, nil
}
func (f *fakeExpenseManager) Delete(_ context.Context, id int64, userID *int64) error {
	f.deletedID, f.deleteUserID = id, userID
	return f.deleteErr
}

type fakeUserManager struct {
	location          *time.Location
	timezone          string
	locationErr       error
	setTimezoneErr    error
	locationUserID    int64
	setTimezoneUserID int64
}

func (f *fakeUserManager) SetTimezone(_ context.Context, userID int64, timezone string) error {
	f.setTimezoneUserID, f.timezone = userID, timezone
	return f.setTimezoneErr
}
func (f *fakeUserManager) Location(_ context.Context, userID int64) (*time.Location, error) {
	f.locationUserID = userID
	if f.locationErr != nil {
		return nil, f.locationErr
	}
	return f.location, nil
}

func newFakeToolHandler() (*ToolHandler, *fakeExpenseManager, *fakeUserManager) {
	expenses := &fakeExpenseManager{}
	users := &fakeUserManager{location: time.UTC}
	return NewToolHandler(expenses, users), expenses, users
}

func TestToolHandlerExecuteRejectsInvalidUserAndUnknownTool(t *testing.T) {
	h, _, _ := newFakeToolHandler()
	if _, err := h.Execute(context.Background(), 0, "list_expenses", `{}`, ""); err == nil {
		t.Fatal("expected invalid user ID error")
	}
	if _, err := h.Execute(context.Background(), 7, "not_a_tool", `{}`, ""); err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("Execute unknown tool error = %v", err)
	}
}

func TestToolHandlerCreateExpense(t *testing.T) {
	h, expenses, users := newFakeToolHandler()
	got, err := h.Execute(context.Background(), 42, "create_expense", `{"amount":35000,"category":"food","comment":"  кофе  ","date":{"type":"unspecified"}}`, "кофе 350")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var created domain.Expense
	if err := json.Unmarshal([]byte(got), &created); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if expenses.createUserID != 42 || created.UserID != 42 || created.Amount != 35000 || created.Category != domain.CategoryFood || created.Comment != "кофе" {
		t.Fatalf("created expense = %#v; manager user ID = %d", created, expenses.createUserID)
	}
	if users.locationUserID != 42 {
		t.Fatalf("Location user ID = %d, want 42", users.locationUserID)
	}
	if created.SpentAt.Location() != time.UTC || created.SpentAt.Hour() != 0 || created.SpentAt.Minute() != 0 {
		t.Fatalf("SpentAt = %v, want midnight UTC", created.SpentAt)
	}
}

func TestToolHandlerCreateExpenseRejectsInvalidArguments(t *testing.T) {
	tests := []struct{ name, args string }{
		{"invalid json", `{`},
		{"zero amount", `{"amount":0,"category":"food","comment":"x"}`},
		{"negative amount", `{"amount":-1,"category":"food","comment":"x"}`},
		{"unknown date type", `{"amount":100,"category":"food","comment":"x","date":{"type":"future"}}`},
		{"relative date without offset", `{"amount":100,"category":"food","comment":"x","date":{"type":"relative"}}`},
		{"invalid calendar date", `{"amount":100,"category":"food","comment":"x","date":{"type":"calendar_date","day":31,"month":2,"year":2025}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, expenses, _ := newFakeToolHandler()
			if _, err := h.Execute(context.Background(), 5, "create_expense", tt.args, ""); err == nil {
				t.Fatal("expected error")
			}
			if expenses.createUserID != 0 {
				t.Fatal("Create should not be called for invalid arguments")
			}
		})
	}
}

func TestToolHandlerListExpensesBuildsCustomDateFilter(t *testing.T) {
	h, expenses, _ := newFakeToolHandler()
	expenses.totalResult = 90000
	expenses.listResult = []domain.Expense{{ID: 1, Amount: 90000}}
	got, err := h.Execute(context.Background(), 9, "list_expenses", `{"period":"custom","from":"2026-10-01","to":"2026-10-03","category":"food"}`, "покажи расходы")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if expenses.listedFilter.UserID == nil || *expenses.listedFilter.UserID != 9 {
		t.Fatalf("UserID filter = %v", expenses.listedFilter.UserID)
	}
	if expenses.listedFilter.Category != domain.CategoryFood {
		t.Fatalf("Category = %q", expenses.listedFilter.Category)
	}
	if expenses.listedFilter.From == nil || expenses.listedFilter.From.Format("2006-01-02") != "2026-10-01" {
		t.Fatalf("From = %v", expenses.listedFilter.From)
	}
	if expenses.listedFilter.To == nil || expenses.listedFilter.To.Format("2006-01-02 15:04:05.999999999") != "2026-10-03 23:59:59.999999999" {
		t.Fatalf("To = %v", expenses.listedFilter.To)
	}
	var result struct {
		Items []domain.Expense `json:"items"`
		Total int64            `json:"total_amount_kopecks"`
	}
	if err := json.Unmarshal([]byte(got), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(result.Items) != 1 || result.Total != 90000 {
		t.Fatalf("result = %#v", result)
	}
}

func TestToolHandlerListExpensesRejectsInvalidPeriodAndRange(t *testing.T) {
	for _, args := range []string{
		`{"period":"unknown"}`,
		`{"period":"last_n_days","days":0}`,
		`{"period":"custom"}`,
		`{"period":"custom","from":"2026-10-04","to":"2026-10-01"}`,
		`{"period":"custom","from":"not-a-date"}`,
	} {
		t.Run(args, func(t *testing.T) {
			h, _, _ := newFakeToolHandler()
			if _, err := h.Execute(context.Background(), 3, "list_expenses", args, ""); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestToolHandlerUpdateExpenseOnlyChangesProvidedFields(t *testing.T) {
	h, expenses, _ := newFakeToolHandler()
	oldDate := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	expenses.gotExpense = domain.Expense{ID: 12, UserID: 4, Amount: 100, Category: domain.CategoryFood, SpentAt: oldDate, Comment: "old"}
	got, err := h.Execute(context.Background(), 4, "update_expense", `{"id":12,"amount":250,"comment":"  lunch  "}`, "измени расход")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if expenses.getID != 12 || expenses.getUserID == nil || *expenses.getUserID != 4 {
		t.Fatalf("GetByID args = id:%d user:%v", expenses.getID, expenses.getUserID)
	}
	if expenses.updated.Amount != 250 || expenses.updated.Comment != "lunch" || !expenses.updated.SpentAt.Equal(oldDate) || expenses.updated.Category != domain.CategoryFood {
		t.Fatalf("updated expense = %#v", expenses.updated)
	}
	if expenses.updateUserID == nil || *expenses.updateUserID != 4 {
		t.Fatalf("Update user ID = %v", expenses.updateUserID)
	}
	if !strings.Contains(got, `"Amount":250`) {
		t.Fatalf("result = %s", got)
	}
}

func TestToolHandlerDeleteExpense(t *testing.T) {
	h, expenses, _ := newFakeToolHandler()
	got, err := h.Execute(context.Background(), 8, "delete_expense", `{"id":55}`, "удали")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got != `{"success":true}` || expenses.deletedID != 55 || expenses.deleteUserID == nil || *expenses.deleteUserID != 8 {
		t.Fatalf("result=%s deletedID=%d user=%v", got, expenses.deletedID, expenses.deleteUserID)
	}
}

func TestToolHandlerSetTimezone(t *testing.T) {
	h, _, users := newFakeToolHandler()
	got, err := h.Execute(context.Background(), 8, "set_timezone", `{"timezone":" Europe/Moscow "}`, "поменяй пояс")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if users.timezone != "Europe/Moscow" || users.setTimezoneUserID != 8 || !strings.Contains(got, `"timezone":"Europe/Moscow"`) {
		t.Fatalf("timezone=%q result=%s", users.timezone, got)
	}
	if _, err := h.Execute(context.Background(), 8, "set_timezone", `{"timezone":"No/Such_Zone"}`, ""); err == nil {
		t.Fatal("expected invalid timezone error")
	}
	if _, err := h.Execute(context.Background(), 8, "set_timezone", `{"timezone":"  "}`, ""); err == nil {
		t.Fatal("expected empty timezone error")
	}
}

func TestToolHandlerWrapsStorageErrors(t *testing.T) {
	h, expenses, users := newFakeToolHandler()
	expected := errors.New("storage unavailable")
	expenses.createErr = expected
	if _, err := h.Execute(context.Background(), 1, "create_expense", `{"amount":100,"category":"food","comment":"x"}`, ""); !errors.Is(err, expected) {
		t.Fatalf("create error = %v, want wrapped storage error", err)
	}
	users.locationErr = expected
	if _, err := h.Execute(context.Background(), 1, "list_expenses", `{"period":"all"}`, ""); !errors.Is(err, expected) {
		t.Fatalf("location error = %v, want wrapped error", err)
	}
}

func TestToolHandlerListExpensesSupportsCalendarPeriods(t *testing.T) {
	today := startOfDay(time.Now().UTC())
	monday := mondayOfWeek(today)
	currentMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	previousMonth := currentMonth.AddDate(0, -1, 0)
	tests := []struct {
		period string
		from   string
		to     string
	}{
		{"all", "", ""},
		{"today", today.Format("2006-01-02"), today.Format("2006-01-02")},
		{"yesterday", today.AddDate(0, 0, -1).Format("2006-01-02"), today.AddDate(0, 0, -1).Format("2006-01-02")},
		{"day_before_yesterday", today.AddDate(0, 0, -2).Format("2006-01-02"), today.AddDate(0, 0, -2).Format("2006-01-02")},
		{"week", monday.Format("2006-01-02"), monday.AddDate(0, 0, 6).Format("2006-01-02")},
		{"previous_week", monday.AddDate(0, 0, -7).Format("2006-01-02"), monday.AddDate(0, 0, -1).Format("2006-01-02")},
		{"last_7_days", today.AddDate(0, 0, -6).Format("2006-01-02"), today.Format("2006-01-02")},
		{"last_n_days", today.AddDate(0, 0, -2).Format("2006-01-02"), today.Format("2006-01-02")},
		{"month", currentMonth.Format("2006-01-02"), currentMonth.AddDate(0, 1, -1).Format("2006-01-02")},
		{"last_month", previousMonth.Format("2006-01-02"), currentMonth.AddDate(0, 0, -1).Format("2006-01-02")},
	}
	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			h, expenses, _ := newFakeToolHandler()
			args := `{"period":"` + tt.period + `"}`
			if tt.period == "last_n_days" {
				args = `{"period":"last_n_days","days":3}`
			}
			if _, err := h.Execute(context.Background(), 23, "list_expenses", args, ""); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			filter := expenses.listedFilter
			if tt.from == "" {
				if filter.From != nil || filter.To != nil {
					t.Fatalf("all period should have no date filter, got %v .. %v", filter.From, filter.To)
				}
				return
			}
			if filter.From == nil || filter.To == nil {
				t.Fatalf("date filter missing: %v .. %v", filter.From, filter.To)
			}
			if filter.From.Format("2006-01-02") != tt.from || filter.To.Format("2006-01-02") != tt.to {
				t.Fatalf("range = %s .. %s, want %s .. %s", filter.From.Format("2006-01-02"), filter.To.Format("2006-01-02"), tt.from, tt.to)
			}
		})
	}
}
