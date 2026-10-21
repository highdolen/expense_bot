package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"expense_tgbot/internal/domain"
)

type fakeUserManager struct {
	users []domain.User
	err   error
}

func (f *fakeUserManager) List(_ context.Context) ([]domain.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.users, nil
}

func TestUserList(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	manager := &fakeUserManager{
		users: []domain.User{
			{
				ID:        1,
				Username:  "test_user",
				FirstName: "Alex",
				Timezone:  "Europe/Berlin",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	handler := NewUserHandler(manager)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d; body: %s",
			rec.Code, http.StatusOK, rec.Body.String())
	}

	var users []UserResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("got %d users, want 1", len(users))
	}
	if users[0].ID != 1 {
		t.Errorf("got user ID %d, want 1", users[0].ID)
	}
	if users[0].Username != "test_user" {
		t.Errorf("got username %q, want test_user", users[0].Username)
	}
	if users[0].Timezone != "Europe/Berlin" {
		t.Errorf("got timezone %q, want Europe/Berlin", users[0].Timezone)
	}
}

func TestUserListServiceError(t *testing.T) {
	handler := NewUserHandler(&fakeUserManager{
		err: errors.New("database unavailable"),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want %d",
			rec.Code, http.StatusInternalServerError)
	}
}
