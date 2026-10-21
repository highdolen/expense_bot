package telegram

import (
	"context"
	"fmt"

	tgbot "github.com/go-telegram/bot"
)

func NewBot(token string, assistant Assistant, users UserService) (*tgbot.Bot, error) {
	if token == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}

	if assistant == nil {
		return nil, fmt.Errorf("assistant is required")
	}

	if users == nil {
		return nil, fmt.Errorf("user service is required")
	}

	handler := NewHandler(assistant, users)

	b, err := tgbot.New(
		token,
		tgbot.WithDefaultHandler(handler.HandleMessage),
	)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	return b, nil
}

func Start(ctx context.Context, b *tgbot.Bot) {
	b.Start(ctx)
}
