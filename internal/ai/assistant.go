package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

const maxHistoryMessages = 20

type AgentProcessor interface {
	Process(
		ctx context.Context,
		userID int64,
		messages []ChatMessage,
	) (ChatMessage, error)
}

type userLock struct {
	mu   sync.Mutex
	refs int
}

type Assistant struct {
	agent AgentProcessor

	// historyMu защищает общую map, но не удерживается во время запроса к LLM.
	historyMu sync.RWMutex
	history   map[int64][]ChatMessage

	// locksMu защищает карту блокировок пользователей.
	locksMu   sync.Mutex
	userLocks map[int64]*userLock
}

func NewAssistant(agent AgentProcessor) *Assistant {
	return &Assistant{
		agent:     agent,
		history:   make(map[int64][]ChatMessage),
		userLocks: make(map[int64]*userLock),
	}
}

func (a *Assistant) ProcessMessage(
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

	// Сообщения одного пользователя обрабатываются последовательно.
	// При этом разные пользователи не блокируют друг друга на время LLM-запроса.
	unlockUser := a.lockUser(userID)
	defer unlockUser()

	a.historyMu.RLock()
	history := append([]ChatMessage(nil), a.history[userID]...)
	a.historyMu.RUnlock()

	history = append(history, ChatMessage{
		Role:    "user",
		Content: message,
	})

	if len(history) > maxHistoryMessages {
		history = history[len(history)-maxHistoryMessages:]
	}

	response, err := a.agent.Process(ctx, userID, history)
	if err != nil {
		return "", fmt.Errorf("process assistant message: %w", err)
	}

	if strings.TrimSpace(response.Content) == "" {
		return "", fmt.Errorf("assistant returned an empty response")
	}

	history = append(history, ChatMessage{
		Role:    "assistant",
		Content: response.Content,
	})

	if len(history) > maxHistoryMessages {
		history = history[len(history)-maxHistoryMessages:]
	}

	a.historyMu.Lock()
	a.history[userID] = history
	a.historyMu.Unlock()

	return response.Content, nil
}

func (a *Assistant) lockUser(userID int64) func() {
	a.locksMu.Lock()
	lock := a.userLocks[userID]
	if lock == nil {
		lock = &userLock{}
		a.userLocks[userID] = lock
	}
	lock.refs++
	a.locksMu.Unlock()

	lock.mu.Lock()

	return func() {
		lock.mu.Unlock()

		a.locksMu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(a.userLocks, userID)
		}
		a.locksMu.Unlock()
	}
}
