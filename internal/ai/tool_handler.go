package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"expense_tgbot/internal/domain"
)

type ExpenseManager interface {
	Create(ctx context.Context, userID int64, expense domain.Expense) (domain.Expense, error)
	List(ctx context.Context, filter domain.ExpenseFilter) ([]domain.Expense, error)
	Total(ctx context.Context, filter domain.ExpenseFilter) (int64, error)
	GetByID(ctx context.Context, id int64, userID *int64) (domain.Expense, error)
	Update(ctx context.Context, expense domain.Expense, userID *int64) (domain.Expense, error)
	Delete(ctx context.Context, id int64, userID *int64) error
}

type UserManager interface {
	SetTimezone(ctx context.Context, userID int64, timezone string) error
	Location(ctx context.Context, userID int64) (*time.Location, error)
}

type ToolHandler struct {
	expenses ExpenseManager
	users    UserManager
}

func NewToolHandler(expenses ExpenseManager, users UserManager) *ToolHandler {
	return &ToolHandler{
		expenses: expenses,
		users:    users,
	}
}

func (h *ToolHandler) Execute(
	ctx context.Context,
	userID int64,
	toolName string,
	arguments string,
	userMessage string,
) (string, error) {
	if userID <= 0 {
		return "", fmt.Errorf("invalid user ID")
	}

	switch toolName {
	case "create_expense":
		return h.createExpense(ctx, userID, arguments)
	case "list_expenses":
		return h.listExpenses(ctx, userID, arguments)
	case "update_expense":
		return h.updateExpense(ctx, userID, arguments)
	case "delete_expense":
		return h.deleteExpense(ctx, userID, arguments)
	case "set_timezone":
		return h.setTimezone(ctx, userID, arguments)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

// expenseDateArgs описывает намерение пользователя относительно даты.
// Конкретную дату вычисляет приложение, а не языковая модель.
type expenseDateArgs struct {
	Type       string `json:"type"`
	OffsetDays *int   `json:"offset_days,omitempty"`
	Day        *int   `json:"day,omitempty"`
	Month      *int   `json:"month,omitempty"`
	Year       *int   `json:"year,omitempty"`
}

type createExpenseArgs struct {
	Amount   int64            `json:"amount"`
	Category domain.Category  `json:"category"`
	Comment  string           `json:"comment"`
	Date     *expenseDateArgs `json:"date"`
}

func (h *ToolHandler) createExpense(
	ctx context.Context,
	userID int64,
	arguments string,
) (string, error) {
	var args createExpenseArgs

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("decode create_expense arguments: %w", err)
	}

	if args.Amount <= 0 {
		return "", fmt.Errorf("expense amount must be positive")
	}

	location, err := h.users.Location(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user timezone: %w", err)
	}

	spentAt, err := resolveStructuredExpenseDate(args.Date, location)
	if err != nil {
		return "", fmt.Errorf("resolve expense date: %w", err)
	}

	expense, err := h.expenses.Create(ctx, userID, domain.Expense{
		Amount:   args.Amount,
		Category: args.Category,
		SpentAt:  spentAt,
		Comment:  strings.TrimSpace(args.Comment),
	})
	if err != nil {
		return "", fmt.Errorf("create expense: %w", err)
	}

	result, err := json.Marshal(expense)
	if err != nil {
		return "", fmt.Errorf("encode created expense: %w", err)
	}

	return string(result), nil
}

func resolveStructuredExpenseDate(
	date *expenseDateArgs,
	location *time.Location,
) (time.Time, error) {
	if location == nil {
		location = time.UTC
	}

	now := time.Now().In(location)
	today := startOfDay(now)

	if date == nil {
		return today, nil
	}

	switch date.Type {
	case "unspecified":
		return today, nil

	case "relative":
		if date.OffsetDays == nil {
			return time.Time{}, fmt.Errorf(
				"offset_days is required for a relative date",
			)
		}

		return today.AddDate(0, 0, *date.OffsetDays), nil

	case "calendar_date":
		if date.Day == nil || date.Month == nil {
			return time.Time{}, fmt.Errorf(
				"day and month are required for a calendar date",
			)
		}

		year := now.Year()
		if date.Year != nil {
			year = *date.Year
		}

		month := time.Month(*date.Month)
		day := *date.Day

		if month < time.January || month > time.December {
			return time.Time{}, fmt.Errorf("invalid month: %d", month)
		}
		if day < 1 || day > 31 {
			return time.Time{}, fmt.Errorf("invalid day: %d", day)
		}
		if year < 1 || year > 9999 {
			return time.Time{}, fmt.Errorf("invalid year: %d", year)
		}

		result := time.Date(year, month, day, 0, 0, 0, 0, location)
		if result.Year() != year ||
			result.Month() != month ||
			result.Day() != day {
			return time.Time{}, fmt.Errorf(
				"invalid calendar date: %04d-%02d-%02d",
				year, month, day,
			)
		}

		return result, nil

	default:
		return time.Time{}, fmt.Errorf("unsupported date type: %s", date.Type)
	}
}

type listExpensesArgs struct {
	Period   string          `json:"period"`
	From     string          `json:"from"`
	To       string          `json:"to"`
	Days     int             `json:"days"`
	Category domain.Category `json:"category"`
}

func (h *ToolHandler) listExpenses(
	ctx context.Context,
	userID int64,
	arguments string,
) (string, error) {
	var args listExpensesArgs

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("decode list_expenses arguments: %w", err)
	}

	location, err := h.users.Location(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user timezone: %w", err)
	}
	if location == nil {
		location = time.UTC
	}

	now := time.Now().In(location)
	today := startOfDay(now)

	var from, to *time.Time

	switch args.Period {
	case "", "all":
		// Все расходы: ограничения по датам отсутствуют.

	case "today":
		start := today
		end := endOfDay(start)
		from, to = &start, &end

	case "yesterday":
		start := today.AddDate(0, 0, -1)
		end := endOfDay(start)
		from, to = &start, &end

	case "day_before_yesterday":
		start := today.AddDate(0, 0, -2)
		end := endOfDay(start)
		from, to = &start, &end

	case "week":
		start := mondayOfWeek(today)
		end := endOfDay(start.AddDate(0, 0, 6))
		from, to = &start, &end

	case "previous_week":
		start := mondayOfWeek(today).AddDate(0, 0, -7)
		end := endOfDay(start.AddDate(0, 0, 6))
		from, to = &start, &end

	case "last_7_days":
		start := today.AddDate(0, 0, -6)
		end := endOfDay(today)
		from, to = &start, &end

	case "last_n_days":
		if args.Days < 1 || args.Days > 3650 {
			return "", fmt.Errorf(
				"days must be between 1 and 3650 for last_n_days",
			)
		}

		start := today.AddDate(0, 0, -(args.Days - 1))
		end := endOfDay(today)
		from, to = &start, &end

	case "month":
		start := time.Date(
			now.Year(), now.Month(), 1,
			0, 0, 0, 0, location,
		)
		end := endOfDay(start.AddDate(0, 1, -1))
		from, to = &start, &end

	case "last_month":
		currentMonth := time.Date(
			now.Year(), now.Month(), 1,
			0, 0, 0, 0, location,
		)
		start := currentMonth.AddDate(0, -1, 0)
		end := endOfDay(currentMonth.AddDate(0, 0, -1))
		from, to = &start, &end

	case "custom":
		if args.From == "" && args.To == "" {
			return "", fmt.Errorf(
				"at least one of from or to is required for custom period",
			)
		}

		if args.From != "" {
			parsed, err := time.ParseInLocation(
				"2006-01-02", args.From, location,
			)
			if err != nil {
				return "", fmt.Errorf("invalid start date: %w", err)
			}

			from = &parsed
		}

		if args.To != "" {
			parsed, err := time.ParseInLocation(
				"2006-01-02", args.To, location,
			)
			if err != nil {
				return "", fmt.Errorf("invalid end date: %w", err)
			}

			end := endOfDay(parsed)
			to = &end
		}

	default:
		return "", fmt.Errorf("unsupported expense period: %s", args.Period)
	}

	if from != nil && to != nil && from.After(*to) {
		return "", fmt.Errorf("start date must not be after end date")
	}

	filter := domain.ExpenseFilter{
		UserID:   &userID,
		Category: args.Category,
		From:     from,
		To:       to,
		Limit:    20,
	}

	expenses, err := h.expenses.List(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("list expenses: %w", err)
	}

	total, err := h.expenses.Total(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("calculate expenses total: %w", err)
	}

	result := struct {
		Items              []domain.Expense `json:"items"`
		TotalAmountKopecks int64            `json:"total_amount_kopecks"`
	}{
		Items:              expenses,
		TotalAmountKopecks: total,
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode expenses: %w", err)
	}

	return string(encoded), nil
}

type updateExpenseArgs struct {
	ID       int64            `json:"id"`
	Amount   *int64           `json:"amount"`
	Category *domain.Category `json:"category"`
	Date     *expenseDateArgs `json:"date"`
	Comment  *string          `json:"comment"`
}

func (h *ToolHandler) updateExpense(
	ctx context.Context,
	userID int64,
	arguments string,
) (string, error) {
	var args updateExpenseArgs

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("decode update_expense arguments: %w", err)
	}

	if args.ID <= 0 {
		return "", fmt.Errorf("invalid expense ID")
	}

	expense, err := h.expenses.GetByID(ctx, args.ID, &userID)
	if err != nil {
		return "", fmt.Errorf("get expense: %w", err)
	}

	if args.Amount != nil {
		if *args.Amount <= 0 {
			return "", fmt.Errorf("expense amount must be positive")
		}
		expense.Amount = *args.Amount
	}

	if args.Category != nil {
		expense.Category = *args.Category
	}

	if args.Date != nil && args.Date.Type != "unspecified" {
		location, err := h.users.Location(ctx, userID)
		if err != nil {
			return "", fmt.Errorf("get user timezone: %w", err)
		}

		spentAt, err := resolveStructuredExpenseDate(args.Date, location)
		if err != nil {
			return "", fmt.Errorf("resolve expense date: %w", err)
		}

		expense.SpentAt = spentAt
	}

	if args.Comment != nil {
		expense.Comment = strings.TrimSpace(*args.Comment)
	}

	updated, err := h.expenses.Update(ctx, expense, &userID)
	if err != nil {
		return "", fmt.Errorf("update expense: %w", err)
	}

	result, err := json.Marshal(updated)
	if err != nil {
		return "", fmt.Errorf("encode updated expense: %w", err)
	}

	return string(result), nil
}

type deleteExpenseArgs struct {
	ID int64 `json:"id"`
}

func (h *ToolHandler) deleteExpense(
	ctx context.Context,
	userID int64,
	arguments string,
) (string, error) {
	var args deleteExpenseArgs

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("decode delete_expense arguments: %w", err)
	}

	if args.ID <= 0 {
		return "", fmt.Errorf("invalid expense ID")
	}

	if err := h.expenses.Delete(ctx, args.ID, &userID); err != nil {
		return "", fmt.Errorf("delete expense: %w", err)
	}

	return `{"success":true}`, nil
}

type setTimezoneArgs struct {
	Timezone string `json:"timezone"`
}

func (h *ToolHandler) setTimezone(
	ctx context.Context,
	userID int64,
	arguments string,
) (string, error) {
	var args setTimezoneArgs

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("decode set_timezone arguments: %w", err)
	}

	args.Timezone = strings.TrimSpace(args.Timezone)
	if args.Timezone == "" {
		return "", fmt.Errorf("timezone must not be empty")
	}

	if _, err := time.LoadLocation(args.Timezone); err != nil {
		return "", fmt.Errorf("invalid timezone %q: %w", args.Timezone, err)
	}

	if err := h.users.SetTimezone(ctx, userID, args.Timezone); err != nil {
		return "", fmt.Errorf("set user timezone: %w", err)
	}

	result, err := json.Marshal(map[string]string{
		"success":  "true",
		"timezone": args.Timezone,
	})
	if err != nil {
		return "", fmt.Errorf("encode timezone result: %w", err)
	}

	return string(result), nil
}

// mondayOfWeek возвращает начало календарной недели — понедельник в 00:00.
func mondayOfWeek(date time.Time) time.Time {
	start := startOfDay(date)
	weekday := (int(start.Weekday()) + 6) % 7
	return start.AddDate(0, 0, -weekday)
}
