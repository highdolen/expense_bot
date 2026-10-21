package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientValidatesAndNormalizesConfig(t *testing.T) {
	for _, values := range [][3]string{{"", "key", "model"}, {"http://localhost", "", "model"}, {"http://localhost", "key", " "}} {
		if _, err := NewClient(values[0], values[1], values[2]); err == nil {
			t.Fatalf("NewClient(%q, %q, %q) expected error", values[0], values[1], values[2])
		}
	}
	client, err := NewClient(" http://localhost/// ", " key ", " model ")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.baseURL != "http://localhost" || client.apiKey != "key" || client.model != "model" {
		t.Fatalf("client config not normalized: %#v", client)
	}
}

func TestClientChatSendsExpectedRequestAndDecodesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		var request chatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.Model != "test-model" || len(request.Messages) != 1 || request.Messages[0].Content != "hello" {
			t.Errorf("request payload = %#v", request)
		}
		if len(request.Tools) != 0 {
			t.Errorf("Chat should not send tools, got %d", len(request.Tools))
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "secret", "test-model")
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if got.Role != "assistant" || got.Content != "hi" {
		t.Fatalf("response = %#v", got)
	}
}

func TestClientChatWithToolsSendsToolDefinitions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request chatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Tools) != len(tools) {
			t.Errorf("tools count = %d, want %d", len(request.Tools), len(tools))
		}
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"done","tool_calls":[{"id":"c1","type":"function","function":{"name":"list_expenses","arguments":"{}"}}]}}]}`); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "key", "model")
	got, err := client.ChatWithTools(context.Background(), []ChatMessage{{Role: "user", Content: "show"}})
	if err != nil {
		t.Fatalf("ChatWithTools: %v", err)
	}
	if len(got.ToolCalls) != 1 || got.ToolCalls[0].Function.Name != "list_expenses" {
		t.Fatalf("response = %#v", got)
	}
}

func TestClientReturnsHTTPAndMalformedResponseErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"provider error", http.StatusServiceUnavailable, "temporarily unavailable", "HTTP status 503"},
		{"invalid json", http.StatusOK, "not-json", "decode LLM response"},
		{"no choices", http.StatusOK, `{"choices":[]}`, "no choices"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				if _, err := fmt.Fprint(w, tt.body); err != nil {
					t.Errorf("failed to write response: %v", err)
				}
			}))
			defer server.Close()
			client, _ := NewClient(server.URL, "key", "model")
			_, err := client.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestClientHonorsCanceledContext(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "key", "model")
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := client.Chat(ctx, []ChatMessage{{Role: "user", Content: "hi"}})
		result <- err
	}()
	<-started
	cancel()
	err := <-result
	close(release)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
