package postgres

import (
	"context"
	"errors"
	"fmt"

	"expense_tgbot/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
	}
}

func (r *UserRepository) GetByID(
	ctx context.Context,
	id int64,
) (domain.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx, `
		SELECT id, username, first_name, timezone, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}

		return domain.User{}, fmt.Errorf("get user by ID: %w", err)
	}

	return user, nil
}

// GetOrCreate регистрирует пользователя или обновляет его данные из Telegram.
func (r *UserRepository) GetOrCreate(
	ctx context.Context,
	user domain.User,
) (domain.User, error) {
	result, err := scanUser(r.pool.QueryRow(ctx, `
		INSERT INTO users (id, username, first_name, timezone)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE
		SET username = EXCLUDED.username,
		    first_name = EXCLUDED.first_name,
		    updated_at = NOW()
		RETURNING id, username, first_name, timezone, created_at, updated_at
	`, user.ID, user.Username, user.FirstName, user.Timezone))
	if err != nil {
		return domain.User{}, fmt.Errorf("get or create user: %w", err)
	}

	return result, nil
}

func (r *UserRepository) UpdateTimezone(
	ctx context.Context,
	id int64,
	timezone string,
) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users
		SET timezone = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, id, timezone)
	if err != nil {
		return fmt.Errorf("update user timezone: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, username, first_name, timezone, created_at, updated_at
		FROM users
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)

	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}

func scanUser(row rowScanner) (domain.User, error) {
	var user domain.User

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.FirstName,
		&user.Timezone,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}
