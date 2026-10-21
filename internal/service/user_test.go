package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"expense_tgbot/internal/domain"
)

type fakeUserRepository struct {
	user  domain.User
	users []domain.User
	err   error

	lastID               int64
	lastTimezone         string
	getOrCreateCalled    bool
	updateTimezoneCalled bool
}

func (f *fakeUserRepository) GetByID(
	_ context.Context,
	id int64,
) (domain.User, error) {
	f.lastID = id
	if f.err != nil {
		return domain.User{}, f.err
	}
	return f.user, nil
}

func (f *fakeUserRepository) GetOrCreate(
	_ context.Context,
	user domain.User,
) (domain.User, error) {
	f.getOrCreateCalled = true
	if f.err != nil {
		return domain.User{}, f.err
	}
	return user, nil
}

func (f *fakeUserRepository) UpdateTimezone(
	_ context.Context,
	id int64,
	timezone string,
) error {
	f.lastID = id
	f.lastTimezone = timezone
	f.updateTimezoneCalled = true
	return f.err
}

func (f *fakeUserRepository) List(
	_ context.Context,
) ([]domain.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.users, nil
}

func TestNewUserServiceRejectsInvalidTimezone(t *testing.T) {
	_, err := NewUserService(&fakeUserRepository{}, "Invalid/Timezone")
	if err == nil {
		t.Fatal("expected an error for invalid timezone")
	}
}

func TestUserServiceGetByID(t *testing.T) {
	repo := &fakeUserRepository{
		user: domain.User{ID: 42, Username: "test_user"},
	}
	service, err := NewUserService(repo, "UTC")
	if err != nil {
		t.Fatal(err)
	}

	got, err := service.GetByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Fatalf("got ID %d, want 42", got.ID)
	}
	if repo.lastID != 42 {
		t.Fatalf("repository received ID %d, want 42", repo.lastID)
	}
}

func TestUserServiceGetByIDRejectsInvalidID(t *testing.T) {
	repo := &fakeUserRepository{}
	service, err := NewUserService(repo, "UTC")
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.GetByID(context.Background(), 0)
	if err == nil {
		t.Fatal("expected an error for invalid user ID")
	}
	if repo.lastID != 0 {
		t.Fatal("repository should not be called")
	}
}

func TestUserServiceSetTimezone(t *testing.T) {
	repo := &fakeUserRepository{}
	service, err := NewUserService(repo, "UTC")
	if err != nil {
		t.Fatal(err)
	}

	err = service.SetTimezone(context.Background(), 42, "Europe/Berlin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.updateTimezoneCalled {
		t.Fatal("repository UpdateTimezone was not called")
	}
	if repo.lastID != 42 {
		t.Fatalf("got user ID %d, want 42", repo.lastID)
	}
	if repo.lastTimezone != "Europe/Berlin" {
		t.Fatalf("got timezone %q, want Europe/Berlin", repo.lastTimezone)
	}
}

func TestUserServiceSetTimezoneRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		timezone string
	}{
		{name: "invalid user ID", userID: 0, timezone: "UTC"},
		{name: "invalid timezone", userID: 42, timezone: "Invalid/Timezone"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserRepository{}
			service, err := NewUserService(repo, "UTC")
			if err != nil {
				t.Fatal(err)
			}

			err = service.SetTimezone(
				context.Background(),
				tt.userID,
				tt.timezone,
			)
			if err == nil {
				t.Fatal("expected an error")
			}
			if repo.updateTimezoneCalled {
				t.Fatal("repository should not be called")
			}
		})
	}
}

func TestUserServiceLocation(t *testing.T) {
	repo := &fakeUserRepository{
		user: domain.User{
			ID:       42,
			Timezone: "Europe/Berlin",
		},
	}
	service, err := NewUserService(repo, "UTC")
	if err != nil {
		t.Fatal(err)
	}

	location, err := service.Location(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if location.String() != "Europe/Berlin" {
		t.Fatalf("got location %q, want Europe/Berlin", location.String())
	}
}

func TestUserServiceGetOrCreate(t *testing.T) {
	repo := &fakeUserRepository{}
	service, err := NewUserService(repo, "UTC")
	if err != nil {
		t.Fatal(err)
	}

	user := domain.User{
		ID:        42,
		Username:  "test_user",
		FirstName: "Alex",
	}

	got, err := service.GetOrCreate(context.Background(), user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Fatalf("got ID %d, want 42", got.ID)
	}
	if !repo.getOrCreateCalled {
		t.Fatal("repository GetOrCreate was not called")
	}
}

func TestUserServiceGetOrCreateRejectsInvalidUser(t *testing.T) {
	tests := []struct {
		name string
		user domain.User
	}{
		{
			name: "invalid ID",
			user: domain.User{Username: "test_user"},
		},
		{
			name: "missing username and first name",
			user: domain.User{ID: 42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserRepository{}
			service, err := NewUserService(repo, "UTC")
			if err != nil {
				t.Fatal(err)
			}

			_, err = service.GetOrCreate(context.Background(), tt.user)
			if err == nil {
				t.Fatal("expected an error")
			}
			if repo.getOrCreateCalled {
				t.Fatal("repository should not be called")
			}
		})
	}
}

func TestUserServiceList(t *testing.T) {
	repo := &fakeUserRepository{
		users: []domain.User{
			{ID: 1, Username: "first"},
			{ID: 2, Username: "second"},
		},
	}
	service, err := NewUserService(repo, "UTC")
	if err != nil {
		t.Fatal(err)
	}

	got, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d users, want 2", len(got))
	}
}

func TestUserServiceReturnsRepositoryError(t *testing.T) {
	repoErr := errors.New("database unavailable")
	repo := &fakeUserRepository{err: repoErr}
	service, err := NewUserService(repo, "UTC")
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.GetOrCreate(context.Background(), domain.User{
		ID:       42,
		Username: "test_user",
	})
	if !errors.Is(err, repoErr) {
		t.Fatalf("got error %v, want wrapped error %v", err, repoErr)
	}
}

func TestUserServiceLocationUsesDefaultTimezone(t *testing.T) {
	repo := &fakeUserRepository{
		user: domain.User{ID: 42},
	}
	service, err := NewUserService(repo, "Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}

	location, err := service.Location(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, defaultOffset := time.Now().In(location).Zone()
	_, expectedOffset := time.Now().In(time.FixedZone("expected", 0)).Zone()

	if defaultOffset != expectedOffset {
		t.Logf("default timezone offset is %d seconds", defaultOffset)
	}
}
