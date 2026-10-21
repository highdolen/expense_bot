package service

import (
	"testing"
)

func TestConversationServiceAddAndGetHistory(t *testing.T) {
	service := NewConversationService()

	err := service.AddMessage(1, ConversationMessage{
		Role:    "user",
		Content: "кофе 350",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = service.AddMessage(1, ConversationMessage{
		Role:    "assistant",
		Content: "Расход добавлен",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := service.GetHistory(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d messages, want 2", len(got))
	}
	if got[0].Content != "кофе 350" {
		t.Fatalf("unexpected first message: %q", got[0].Content)
	}
	if got[1].Role != "assistant" {
		t.Fatalf("got role %q, want assistant", got[1].Role)
	}
}

func TestConversationServiceLimitsHistory(t *testing.T) {
	service := NewConversationService()

	for i := 0; i < 25; i++ {
		err := service.AddMessage(1, ConversationMessage{
			Role:    "user",
			Content: string(rune('A' + i)),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	got, err := service.GetHistory(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != maxConversationMessages {
		t.Fatalf("got %d messages, want %d", len(got), maxConversationMessages)
	}

	if got[0].Content != string(rune('A'+5)) {
		t.Fatalf("oldest retained message is %q", got[0].Content)
	}
}

func TestConversationServiceKeepsUsersSeparate(t *testing.T) {
	service := NewConversationService()

	_ = service.AddMessage(1, ConversationMessage{
		Role:    "user",
		Content: "первый пользователь",
	})
	_ = service.AddMessage(2, ConversationMessage{
		Role:    "user",
		Content: "второй пользователь",
	})

	got, err := service.GetHistory(1)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0].Content != "первый пользователь" {
		t.Fatalf("unexpected history for user 1: %+v", got)
	}
}

func TestConversationServiceRejectsInvalidInput(t *testing.T) {
	service := NewConversationService()

	if err := service.AddMessage(0, ConversationMessage{Role: "user"}); err == nil {
		t.Fatal("expected an error for invalid user ID")
	}

	if err := service.AddMessage(1, ConversationMessage{Role: "system"}); err == nil {
		t.Fatal("expected an error for invalid role")
	}

	if _, err := service.GetHistory(0); err == nil {
		t.Fatal("expected an error for invalid user ID")
	}

	if err := service.ClearHistory(0); err == nil {
		t.Fatal("expected an error for invalid user ID")
	}
}

func TestConversationServiceClearHistory(t *testing.T) {
	service := NewConversationService()

	if err := service.AddMessage(1, ConversationMessage{
		Role:    "user",
		Content: "кофе 350",
	}); err != nil {
		t.Fatal(err)
	}

	if err := service.ClearHistory(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := service.GetHistory(1)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 0 {
		t.Fatalf("got %d messages after clearing, want 0", len(got))
	}
}

func TestConversationServiceReturnsCopyOfHistory(t *testing.T) {
	service := NewConversationService()

	if err := service.AddMessage(1, ConversationMessage{
		Role:    "user",
		Content: "original",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := service.GetHistory(1)
	if err != nil {
		t.Fatal(err)
	}

	got[0].Content = "changed"

	original, err := service.GetHistory(1)
	if err != nil {
		t.Fatal(err)
	}

	if original[0].Content != "original" {
		t.Fatalf("stored history was modified: %q", original[0].Content)
	}
}
