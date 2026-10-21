package telegram

import "context"

type Assistant interface {
	HandleMessage(ctx context.Context, userID int64, message string) (string, error)
}
