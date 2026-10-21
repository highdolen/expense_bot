package service

import (
	"context"
	"fmt"
	"strings"
)

type AssistantAgent interface {
	ProcessMessage(
		ctx context.Context,
		userID int64,
		message string,
	) (string, error)
}

type AssistantService struct {
	agent AssistantAgent
}

func NewAssistantService(agent AssistantAgent) *AssistantService {
	return &AssistantService{
		agent: agent,
	}
}

func (s *AssistantService) HandleMessage(
	ctx context.Context,
	userID int64,
	message string,
) (string, error) {
	if userID <= 0 {
		return "", fmt.Errorf("invalid user ID")
	}

	message = strings.TrimSpace(message)
	if message == "" {
		return "", fmt.Errorf("message must not be empty")
	}

	if len(message) > 1000 {
		return "", fmt.Errorf("message is too long")
	}

	return s.agent.ProcessMessage(ctx, userID, message)
}
