package service

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeAssistantAgent struct {
	userID   int64
	message  string
	called   bool
	response string
	err      error
}

func (f *fakeAssistantAgent) ProcessMessage(
	_ context.Context,
	userID int64,
	message string,
) (string, error) {
	f.called = true
	f.userID = userID
	f.message = message
	return f.response, f.err
}

func TestAssistantServiceHandleMessage(t *testing.T) {
	agent := &fakeAssistantAgent{response: "Расход добавлен"}
	service := NewAssistantService(agent)

	got, err := service.HandleMessage(
		context.Background(),
		42,
		"  кофе 350  ",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "Расход добавлен" {
		t.Fatalf("got %q, want %q", got, "Расход добавлен")
	}
	if !agent.called {
		t.Fatal("agent was not called")
	}
	if agent.userID != 42 {
		t.Fatalf("got user ID %d, want 42", agent.userID)
	}
	if agent.message != "кофе 350" {
		t.Fatalf("got message %q, want %q", agent.message, "кофе 350")
	}
}

func TestAssistantServiceRejectsInvalidMessages(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		message string
	}{
		{name: "invalid user ID", userID: 0, message: "кофе 350"},
		{name: "empty message", userID: 42, message: "   "},
		{name: "message too long", userID: 42, message: strings.Repeat("a", 1001)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &fakeAssistantAgent{}
			service := NewAssistantService(agent)

			_, err := service.HandleMessage(
				context.Background(),
				tt.userID,
				tt.message,
			)
			if err == nil {
				t.Fatal("expected an error")
			}
			if agent.called {
				t.Fatal("agent should not be called for invalid input")
			}
		})
	}
}

func TestAssistantServiceReturnsAgentError(t *testing.T) {
	agentErr := errors.New("AI service unavailable")
	agent := &fakeAssistantAgent{err: agentErr}
	service := NewAssistantService(agent)

	_, err := service.HandleMessage(context.Background(), 42, "кофе 350")
	if !errors.Is(err, agentErr) {
		t.Fatalf("got error %v, want %v", err, agentErr)
	}
}
