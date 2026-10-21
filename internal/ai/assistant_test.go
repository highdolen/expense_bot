package ai

import (
	"context"
	"sync"
	"testing"
	"time"
)

type blockingAgent struct {
	entered chan int64
	release chan struct{}
}

func (a *blockingAgent) Process(
	_ context.Context,
	userID int64,
	_ []ChatMessage,
) (ChatMessage, error) {
	a.entered <- userID
	<-a.release
	return ChatMessage{Role: "assistant", Content: "готово"}, nil
}

func TestAssistantProcessesDifferentUsersConcurrently(t *testing.T) {
	agent := &blockingAgent{
		entered: make(chan int64, 2),
		release: make(chan struct{}, 2),
	}
	assistant := NewAssistant(agent)

	var wg sync.WaitGroup
	for _, userID := range []int64{1, 2} {
		wg.Add(1)
		go func(userID int64) {
			defer wg.Done()
			if _, err := assistant.ProcessMessage(
				context.Background(),
				userID,
				"кофе 350",
			); err != nil {
				t.Errorf("ProcessMessage(%d): %v", userID, err)
			}
		}(userID)
	}

	seen := make(map[int64]bool)
	for range 2 {
		select {
		case userID := <-agent.entered:
			seen[userID] = true
		case <-time.After(time.Second):
			t.Fatal("different users were serialized while waiting for the agent")
		}
	}

	agent.release <- struct{}{}
	agent.release <- struct{}{}
	wg.Wait()

	if !seen[1] || !seen[2] {
		t.Fatalf("agent entered for users %v, want users 1 and 2", seen)
	}
}

func TestAssistantSerializesMessagesForSameUser(t *testing.T) {
	agent := &blockingAgent{
		entered: make(chan int64, 2),
		release: make(chan struct{}, 2),
	}
	assistant := NewAssistant(agent)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := assistant.ProcessMessage(
			context.Background(),
			1,
			"кофе 350",
		); err != nil {
			t.Errorf("first ProcessMessage: %v", err)
		}
	}()

	select {
	case <-agent.entered:
	case <-time.After(time.Second):
		t.Fatal("first message did not reach the agent")
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := assistant.ProcessMessage(
			context.Background(),
			1,
			"такси 500",
		); err != nil {
			t.Errorf("second ProcessMessage: %v", err)
		}
	}()

	select {
	case <-agent.entered:
		t.Fatal("second message for the same user entered before the first finished")
	case <-time.After(100 * time.Millisecond):
	}

	agent.release <- struct{}{}

	select {
	case <-agent.entered:
	case <-time.After(time.Second):
		t.Fatal("second message did not enter after the first finished")
	}

	agent.release <- struct{}{}
	wg.Wait()
}
