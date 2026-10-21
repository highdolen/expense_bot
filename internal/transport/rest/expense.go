package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"expense_tgbot/internal/domain"
)

type ExpenseManager interface {
	Create(ctx context.Context, userID int64, expense domain.Expense) (domain.Expense, error)
	GetByID(ctx context.Context, id int64, userID *int64) (domain.Expense, error)
	List(ctx context.Context, filter domain.ExpenseFilter) ([]domain.Expense, error)
	Total(ctx context.Context, filter domain.ExpenseFilter) (int64, error)
	Update(ctx context.Context, expense domain.Expense, userID *int64) (domain.Expense, error)
	Delete(ctx context.Context, id int64, userID *int64) error
}

type ExpenseHandler struct {
	expenses ExpenseManager
}

func NewExpenseHandler(expenses ExpenseManager) *ExpenseHandler {
	return &ExpenseHandler{expenses: expenses}
}

type expensesResponse struct {
	Items []ExpenseResponse `json:"items"`
	Total Money             `json:"total"`
}

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filter := domain.ExpenseFilter{
		Limit:  20,
		Offset: 0,
	}

	if value := query.Get("user_id"); value != "" {
		userID, err := strconv.ParseInt(value, 10, 64)
		if err != nil || userID <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_user_id", "Некорректный user_id")
			return
		}
		filter.UserID = &userID
	}

	if value := query.Get("category"); value != "" {
		category := domain.Category(value)
		if !category.IsValid() {
			writeError(w, http.StatusBadRequest, "invalid_category", "Неизвестная категория расходов")
			return
		}
		filter.Category = category
	}

	if value := query.Get("from"); value != "" {
		from, err := time.Parse("2006-01-02", value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_date", "Параметр from должен иметь формат YYYY-MM-DD")
			return
		}
		filter.From = &from
	}

	if value := query.Get("to"); value != "" {
		to, err := time.Parse("2006-01-02", value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_date", "Параметр to должен иметь формат YYYY-MM-DD")
			return
		}
		filter.To = &to
	}

	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		writeError(w, http.StatusBadRequest, "invalid_date_range", "Дата начала периода не может быть позже даты окончания")
		return
	}

	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 {
			writeError(w, http.StatusBadRequest, "invalid_limit", "Параметр limit должен быть от 1 до 100")
			return
		}
		filter.Limit = limit
	}

	if value := query.Get("offset"); value != "" {
		offset, err := strconv.Atoi(value)
		if err != nil || offset < 0 {
			writeError(w, http.StatusBadRequest, "invalid_offset", "Параметр offset должен быть неотрицательным числом")
			return
		}
		filter.Offset = offset
	}

	expenses, err := h.expenses.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить расходы")
		return
	}

	total, err := h.expenses.Total(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось посчитать сумму расходов")
		return
	}

	items := make([]ExpenseResponse, 0, len(expenses))
	for _, expense := range expenses {
		items = append(items, expenseToResponse(expense))
	}

	writeJSON(w, http.StatusOK, expensesResponse{
		Items: items,
		Total: Money(total),
	})
}

func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request CreateExpenseRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Некорректное тело запроса")
		return
	}

	if request.UserID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_user_id", "Необходимо указать корректный user_id")
		return
	}

	if request.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_amount", "Сумма должна быть больше нуля")
		return
	}

	category := domain.Category(strings.TrimSpace(request.Category))
	if !category.IsValid() {
		writeError(w, http.StatusBadRequest, "invalid_category", "Неизвестная категория расходов")
		return
	}

	spentAt, err := time.Parse("2006-01-02", request.SpentAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_date", "Дата должна иметь формат YYYY-MM-DD")
		return
	}

	expense := domain.Expense{
		UserID:   request.UserID,
		Amount:   int64(request.Amount),
		Category: category,
		SpentAt:  spentAt,
		Comment:  strings.TrimSpace(request.Comment),
	}

	created, err := h.expenses.Create(r.Context(), request.UserID, expense)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, expenseToResponse(created))
}

func (h *ExpenseHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := expenseIDFromRequest(w, r)
	if !ok {
		return
	}

	var request UpdateExpenseRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Некорректное тело запроса")
		return
	}

	// nil означает, что веб-панель работает со всеми расходами приложения.
	expense, err := h.expenses.GetByID(r.Context(), id, nil)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	if request.Amount != nil {
		if *request.Amount <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_amount", "Сумма должна быть больше нуля")
			return
		}
		expense.Amount = int64(*request.Amount)
	}

	if request.Category != nil {
		category := domain.Category(strings.TrimSpace(*request.Category))
		if !category.IsValid() {
			writeError(w, http.StatusBadRequest, "invalid_category", "Неизвестная категория расходов")
			return
		}
		expense.Category = category
	}

	if request.SpentAt != nil {
		spentAt, err := time.Parse("2006-01-02", *request.SpentAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_date", "Дата должна иметь формат YYYY-MM-DD")
			return
		}
		expense.SpentAt = spentAt
	}

	if request.Comment != nil {
		expense.Comment = strings.TrimSpace(*request.Comment)
	}

	updated, err := h.expenses.Update(r.Context(), expense, nil)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, expenseToResponse(updated))
}

func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := expenseIDFromRequest(w, r)
	if !ok {
		return
	}

	if err := h.expenses.Delete(r.Context(), id, nil); err != nil {
		h.writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ExpenseHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrExpenseNotFound):
		writeError(w, http.StatusNotFound, "expense_not_found", "Расход не найден")
	case errors.Is(err, domain.ErrUserNotFound):
		writeError(w, http.StatusBadRequest, "user_not_found", "Пользователь не найден")
	case errors.Is(err, domain.ErrInvalidAmount):
		writeError(w, http.StatusBadRequest, "invalid_amount", "Некорректная сумма")
	case errors.Is(err, domain.ErrInvalidCategory):
		writeError(w, http.StatusBadRequest, "invalid_category", "Неизвестная категория расходов")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить операцию")
	}
}

func expenseIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_expense_id", "Некорректный идентификатор расхода")
		return 0, false
	}

	return id, true
}

func expenseToResponse(expense domain.Expense) ExpenseResponse {
	return ExpenseResponse{
		ID:        expense.ID,
		UserID:    expense.UserID,
		Amount:    Money(expense.Amount),
		Category:  string(expense.Category),
		SpentAt:   expense.SpentAt.Format("2006-01-02"),
		Comment:   expense.Comment,
		CreatedAt: expense.CreatedAt,
		UpdatedAt: expense.UpdatedAt,
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}()

	const maxBodySize = 1 << 20 // 1 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request contains more than one JSON value")
		}
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{
		Error: apiError{
			Code:    code,
			Message: message,
		},
	})
}
