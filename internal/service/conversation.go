package service

import (
	"fmt"
	"sync"
)

const maxConversationMessages = 20

type ConversationMessage struct {
	Role    string
	Content string
}

type ConversationService struct {
	mu      sync.RWMutex
	history map[int64][]ConversationMessage
}

func NewConversationService() *ConversationService {
	return &ConversationService{
		history: make(map[int64][]ConversationMessage),
	}
}

func (s *ConversationService) AddMessage(
	userID int64,
	message ConversationMessage,
) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	if message.Role != "user" && message.Role != "assistant" {
		return fmt.Errorf("invalid message role")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	messages := append(s.history[userID], message)

	if len(messages) > maxConversationMessages {
		messages = messages[len(messages)-maxConversationMessages:]
	}

	s.history[userID] = messages

	return nil
}

func (s *ConversationService) GetHistory(
	userID int64,
) ([]ConversationMessage, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Возвращаем копию, чтобы вызывающий код не изменил историю напрямую.
	messages := append([]ConversationMessage(nil), s.history[userID]...)

	return messages, nil
}

func (s *ConversationService) ClearHistory(userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.history, userID)

	return nil
}
