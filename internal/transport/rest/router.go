package rest

import (
	"net/http"

	"expense_tgbot/internal/transport/rest/middleware"
)

type Handlers struct {
	Expenses *ExpenseHandler
	Users    *UserHandler
}

func NewRouter(handlers Handlers) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/expenses", handlers.Expenses.List)
	mux.HandleFunc("POST /api/expenses", handlers.Expenses.Create)
	mux.HandleFunc("PUT /api/expenses/{id}", handlers.Expenses.Update)
	mux.HandleFunc("DELETE /api/expenses/{id}", handlers.Expenses.Delete)

	mux.HandleFunc("GET /api/users", handlers.Users.List)

	// Фронтенд
	mux.Handle("GET /", NewWebHandler())

	return middleware.Recovery(middleware.Logging(mux))
}
