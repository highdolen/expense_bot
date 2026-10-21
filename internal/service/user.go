package service

import (
	"context"
	"fmt"
	"time"

	"expense_tgbot/internal/domain"
)

type UserRepository interface {
	GetByID(ctx context.Context, id int64) (domain.User, error)
	GetOrCreate(ctx context.Context, user domain.User) (domain.User, error)
	UpdateTimezone(ctx context.Context, id int64, timezone string) error
	List(ctx context.Context) ([]domain.User, error)
}

type UserService struct {
	repo            UserRepository
	defaultLocation *time.Location
}

func NewUserService(
	repo UserRepository,
	defaultTimezone string,
) (*UserService, error) {
	location, err := time.LoadLocation(defaultTimezone)
	if err != nil {
		return nil, fmt.Errorf("load default timezone: %w", err)
	}

	return &UserService{
		repo:            repo,
		defaultLocation: location,
	}, nil
}

func (s *UserService) GetByID(
	ctx context.Context,
	id int64,
) (domain.User, error) {
	if id <= 0 {
		return domain.User{}, fmt.Errorf("invalid user ID")
	}

	return s.repo.GetByID(ctx, id)
}

func (s *UserService) SetTimezone(
	ctx context.Context,
	userID int64,
	timezone string,
) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return domain.ErrInvalidTimezone
	}

	return s.repo.UpdateTimezone(ctx, userID, location.String())
}

func (s *UserService) Location(
	ctx context.Context,
	userID int64,
) (*time.Location, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user.Location(s.defaultLocation), nil
}

func (s *UserService) GetOrCreate(
	ctx context.Context,
	user domain.User,
) (domain.User, error) {
	if user.ID <= 0 {
		return domain.User{}, fmt.Errorf("invalid user ID")
	}

	if user.Username == "" && user.FirstName == "" {
		return domain.User{}, fmt.Errorf("username or first name must be provided")
	}

	result, err := s.repo.GetOrCreate(ctx, user)
	if err != nil {
		return domain.User{}, fmt.Errorf("get or create user: %w", err)
	}

	return result, nil
}

func (s *UserService) List(ctx context.Context) ([]domain.User, error) {
	users, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	return users, nil
}
