package rest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expense_tgbot/internal/domain"
)

type routerUserManager struct {
	err error
}

func (f *routerUserManager) List(_ context.Context) ([]domain.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []domain.User{}, nil
}

func TestRouterRoutes(t *testing.T) {
	expenses := &fakeExpenseManager{}
	users := &routerUserManager{}

	router := NewRouter(Handlers{
		Expenses: NewExpenseHandler(expenses),
		Users:    NewUserHandler(users),
	})

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "list expenses",
			method:     http.MethodGet,
			path:       "/api/expenses",
			wantStatus: http.StatusOK,
		},
		{
			name:       "list users",
			method:     http.MethodGet,
			path:       "/api/users",
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       "/api/unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("got status %d, want %d; body: %s",
					rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestRouterServesFrontend(t *testing.T) {
	router := NewRouter(Handlers{
		Expenses: NewExpenseHandler(&fakeExpenseManager{}),
		Users:    NewUserHandler(&routerUserManager{}),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	if !strings.Contains(strings.ToLower(rec.Body.String()), "<html") {
		t.Fatal("expected the response to contain HTML")
	}
}

func TestRouterReturnsInternalErrorForUsers(t *testing.T) {
	router := NewRouter(Handlers{
		Expenses: NewExpenseHandler(&fakeExpenseManager{}),
		Users: NewUserHandler(&routerUserManager{
			err: errors.New("database unavailable"),
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want %d",
			rec.Code, http.StatusInternalServerError)
	}
}
