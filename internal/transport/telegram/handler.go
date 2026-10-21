package telegram

import (
	"context"
	"log"
	"strings"

	"expense_tgbot/internal/domain"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type UserService interface {
	GetOrCreate(ctx context.Context, user domain.User) (domain.User, error)
}

type Handler struct {
	assistant Assistant
	users     UserService
}

func NewHandler(assistant Assistant, users UserService) *Handler {
	return &Handler{
		assistant: assistant,
		users:     users,
	}
}

func (h *Handler) HandleMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	message := update.Message
	userID := message.From.ID
	chatID := message.Chat.ID

	text := strings.TrimSpace(message.Text)
	if text == "" {
		h.sendMessage(ctx, b, chatID, "Пожалуйста, отправь текстовое сообщение.")
		return
	}

	// Регистрируем пользователя перед обработкой любого сообщения.
	user := domain.User{
		ID:        userID,
		Username:  message.From.Username,
		FirstName: message.From.FirstName,
	}

	if _, err := h.users.GetOrCreate(ctx, user); err != nil {
		log.Printf("failed to register user %d: %v", userID, err)
		h.sendMessage(
			ctx,
			b,
			chatID,
			"Не удалось зарегистрировать пользователя. Попробуй ещё раз.",
		)
		return
	}

	parts := strings.Fields(text)
	command := parts[0]

	switch {
	case command == "/start" || strings.HasPrefix(command, "/start@"):
		h.sendMessage(
			ctx,
			b,
			chatID,
			"Привет! Я помогу вести учёт расходов.\n\n"+
				"Напиши, например: «кофе 350» или «продукты 1200».\n"+
				"Также можешь попросить показать расходы за неделю или изменить запись.",
		)
		return
	}

	response, err := h.assistant.HandleMessage(ctx, userID, text)
	if err != nil {
		log.Printf("failed to process message from user %d: %v", userID, err)
		h.sendMessage(ctx, b, chatID, "Не удалось обработать сообщение. Попробуй ещё раз.")
		return
	}

	h.sendMessage(ctx, b, chatID, response)
}

func (h *Handler) sendMessage(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		log.Printf("failed to send Telegram message: %v", err)
	}
}
