package rest

import (
	"context"
	"net/http"

	"expense_tgbot/internal/domain"
)

type UserManager interface {
	List(ctx context.Context) ([]domain.User, error)
}

type UserHandler struct {
	users UserManager
}

func NewUserHandler(users UserManager) *UserHandler {
	return &UserHandler{
		users: users,
	}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"Не удалось получить список пользователей",
		)
		return
	}

	items := make([]UserResponse, 0, len(users))

	for _, user := range users {
		items = append(items, UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			FirstName: user.FirstName,
			Timezone:  user.Timezone,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, items)
}
