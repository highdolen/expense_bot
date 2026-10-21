package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"expense_tgbot/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ExpenseRepository struct {
	pool *pgxpool.Pool
}

func NewExpenseRepository(pool *pgxpool.Pool) *ExpenseRepository {
	return &ExpenseRepository{pool: pool}
}

func (r *ExpenseRepository) Create(
	ctx context.Context,
	expense domain.Expense,
) (domain.Expense, error) {
	created, err := scanExpense(r.pool.QueryRow(ctx, `
		INSERT INTO expenses (
			user_id, amount, category, spent_at, comment
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, amount, category, spent_at, comment,
		          created_at, updated_at
	`,
		expense.UserID,
		expense.Amount,
		expense.Category,
		expense.SpentAt,
		expense.Comment,
	))
	if err != nil {
		return domain.Expense{}, fmt.Errorf("create expense: %w", err)
	}

	return created, nil
}

func (r *ExpenseRepository) GetByID(
	ctx context.Context,
	id int64,
	userID *int64,
) (domain.Expense, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, user_id, amount, category, spent_at, comment,
		       created_at, updated_at
		FROM expenses
		WHERE id = $1
	`)

	args := []any{id}

	if userID != nil {
		args = append(args, *userID)
		fmt.Fprintf(&query, " AND user_id = $%d", len(args))
	}

	expense, err := scanExpense(r.pool.QueryRow(ctx, query.String(), args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Expense{}, domain.ErrExpenseNotFound
		}

		return domain.Expense{}, fmt.Errorf("get expense by ID: %w", err)
	}

	return expense, nil
}

func (r *ExpenseRepository) List(
	ctx context.Context,
	filter domain.ExpenseFilter,
) ([]domain.Expense, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, user_id, amount, category, spent_at, comment,
		       created_at, updated_at
		FROM expenses
	`)

	where, args := buildExpenseWhere(filter)
	query.WriteString(where)

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	args = append(args, limit)
	fmt.Fprintf(&query, " ORDER BY spent_at DESC, id DESC LIMIT $%d", len(args))

	args = append(args, offset)
	fmt.Fprintf(&query, " OFFSET $%d", len(args))

	rows, err := r.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()

	expenses := make([]domain.Expense, 0)

	for rows.Next() {
		expense, err := scanExpense(rows)
		if err != nil {
			return nil, fmt.Errorf("scan expense: %w", err)
		}

		expenses = append(expenses, expense)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expenses: %w", err)
	}

	return expenses, nil
}

func (r *ExpenseRepository) Total(
	ctx context.Context,
	filter domain.ExpenseFilter,
) (int64, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT COALESCE(SUM(amount), 0)
		FROM expenses
	`)

	where, args := buildExpenseWhere(filter)
	query.WriteString(where)

	var total int64
	if err := r.pool.QueryRow(ctx, query.String(), args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("calculate expenses total: %w", err)
	}

	return total, nil
}

func (r *ExpenseRepository) Update(
	ctx context.Context,
	expense domain.Expense,
	userID *int64,
) (domain.Expense, error) {
	query := strings.Builder{}
	query.WriteString(`
		UPDATE expenses
		SET amount = $2,
		    category = $3,
		    spent_at = $4,
		    comment = $5,
		    updated_at = NOW()
		WHERE id = $1
	`)

	args := []any{
		expense.ID,
		expense.Amount,
		expense.Category,
		expense.SpentAt,
		expense.Comment,
	}

	if userID != nil {
		args = append(args, *userID)
		fmt.Fprintf(&query, " AND user_id = $%d", len(args))
	}

	query.WriteString(`
		RETURNING id, user_id, amount, category, spent_at, comment,
		          created_at, updated_at
	`)

	updated, err := scanExpense(r.pool.QueryRow(ctx, query.String(), args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Expense{}, domain.ErrExpenseNotFound
		}

		return domain.Expense{}, fmt.Errorf("update expense: %w", err)
	}

	return updated, nil
}

func (r *ExpenseRepository) Delete(
	ctx context.Context,
	id int64,
	userID *int64,
) error {
	query := strings.Builder{}
	query.WriteString("DELETE FROM expenses WHERE id = $1")

	args := []any{id}

	if userID != nil {
		args = append(args, *userID)
		fmt.Fprintf(&query, " AND user_id = $%d", len(args))
	}

	tag, err := r.pool.Exec(ctx, query.String(), args...)
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrExpenseNotFound
	}

	return nil
}

// buildExpenseWhere собирает общие фильтры для List и Total.
func buildExpenseWhere(filter domain.ExpenseFilter) (string, []any) {
	query := strings.Builder{}
	query.WriteString(" WHERE TRUE")

	args := make([]any, 0, 4)

	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		fmt.Fprintf(&query, " AND expenses.user_id = $%d", len(args))
	}

	if filter.Category != "" {
		args = append(args, filter.Category)
		fmt.Fprintf(&query, " AND expenses.category = $%d", len(args))
	}

	if filter.From != nil {
		args = append(args, *filter.From)
		fmt.Fprintf(&query, " AND expenses.spent_at >= $%d", len(args))
	}

	if filter.To != nil {
		args = append(args, *filter.To)
		fmt.Fprintf(&query, " AND expenses.spent_at <= $%d", len(args))
	}

	return query.String(), args
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanExpense(row rowScanner) (domain.Expense, error) {
	var expense domain.Expense
	var category string

	err := row.Scan(
		&expense.ID,
		&expense.UserID,
		&expense.Amount,
		&category,
		&expense.SpentAt,
		&expense.Comment,
		&expense.CreatedAt,
		&expense.UpdatedAt,
	)
	if err != nil {
		return domain.Expense{}, err
	}

	expense.Category = domain.Category(category)

	return expense, nil
}
