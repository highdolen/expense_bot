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

type recordingToolExecutor struct {
	calls   []string
	results map[string]string
	err     error
}

func (f *recordingToolExecutor) Execute(_ context.Context, userID int64, name, args, message string) (string, error) {
	f.calls = append(f.calls, fmt.Sprintf("%d:%s:%s:%s", userID, name, args, message))
	if f.err != nil {
		return "", f.err
	}
	if result, ok := f.results[name]; ok {
		return result, nil
	}
	return `{"ok":true}`, nil
}

func writeLLMResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if _, err := fmt.Fprint(w, body); err != nil {
		t.Errorf("failed to write response: %v", err)
	}
}

func TestAgentExecutesToolAndReturnsFinalResponse(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var request chatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if requestCount == 1 {
			if len(request.Tools) == 0 {
				t.Error("first request should include tools")
			}
			writeLLMResponse(t, w, `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call-1","type":"function","function":{"name":"create_expense","arguments":"{\"amount\":100}"}}]}}]}`)
			return
		}
		if len(request.Messages) < 4 {
			t.Errorf("second request messages = %d, want system/user/assistant/tool", len(request.Messages))
		}
		writeLLMResponse(t, w, `{"choices":[{"message":{"role":"assistant","content":"Расход добавлен"}}]}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "key", "model")
	executor := &recordingToolExecutor{results: map[string]string{"create_expense": `{"ID":1}`}}
	agent := NewAgent(client, executor)
	got, err := agent.Process(context.Background(), 17, []ChatMessage{{Role: "user", Content: "кофе 100"}})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if got.Content != "Расход добавлен" || requestCount != 2 || len(executor.calls) != 1 {
		t.Fatalf("response=%#v requests=%d calls=%v", got, requestCount, executor.calls)
	}
	if !strings.Contains(executor.calls[0], "17:create_expense") || !strings.Contains(executor.calls[0], "кофе 100") {
		t.Fatalf("tool call context = %q", executor.calls[0])
	}
}

func TestAgentDoesNotExecuteIdenticalToolCallTwice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request chatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Messages) == 2 {
			writeLLMResponse(t, w, `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"first","type":"function","function":{"name":"list_expenses","arguments":"{}"}},{"id":"second","type":"function","function":{"name":"list_expenses","arguments":"{}"}}]}}]}`)
			return
		}
		if request.Messages[len(request.Messages)-1].Role != "tool" {
			t.Errorf("last message role = %q", request.Messages[len(request.Messages)-1].Role)
		}
		writeLLMResponse(t, w, `{"choices":[{"message":{"role":"assistant","content":"Готово"}}]}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "key", "model")
	executor := &recordingToolExecutor{}
	agent := NewAgent(client, executor)
	_, err := agent.Process(context.Background(), 2, []ChatMessage{{Role: "user", Content: "покажи расходы"}})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if len(executor.calls) != 1 {
		t.Fatalf("Execute called %d times, want 1", len(executor.calls))
	}
}

func TestAgentReturnsToolErrorsToModelAndStopsAfterMaxRounds(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var request chatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Tools) > 0 {
			if requestCount > maxToolCallRounds {
				t.Errorf("unexpected tool-enabled request %d", requestCount)
			}
			writeLLMResponse(t, w, fmt.Sprintf(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call-%d","type":"function","function":{"name":"bad_tool","arguments":"{\\\"round\\\":%d}"}}]}}]}`, requestCount, requestCount))
			return
		}
		if len(request.Tools) != 0 {
			t.Error("final request after max rounds must not include tools")
		}
		last := request.Messages[len(request.Messages)-1]
		if last.Role != "tool" || !strings.Contains(last.Content, "operation_failed") {
			t.Errorf("last tool result = %#v", last)
		}
		writeLLMResponse(t, w, `{"choices":[{"message":{"role":"assistant","content":"Не удалось выполнить операцию"}}]}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "key", "model")
	executor := &recordingToolExecutor{err: fmt.Errorf("tool failed")}
	agent := NewAgent(client, executor)
	got, err := agent.Process(context.Background(), 5, []ChatMessage{{Role: "user", Content: "сделай"}})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if got.Content == "" || requestCount != maxToolCallRounds+1 || len(executor.calls) != maxToolCallRounds {
		t.Fatalf("response=%#v requests=%d tool calls=%d", got, requestCount, len(executor.calls))
	}
}

func TestAgentValidatesInputAndPropagatesClientError(t *testing.T) {
	client, _ := NewClient("http://127.0.0.1:1", "key", "model")
	agent := NewAgent(client, &recordingToolExecutor{})
	if _, err := agent.Process(context.Background(), 0, []ChatMessage{{Role: "user", Content: "hi"}}); err == nil {
		t.Fatal("expected invalid user error")
	}
	if _, err := agent.Process(context.Background(), 1, nil); err == nil {
		t.Fatal("expected empty messages error")
	}
	if _, err := agent.Process(context.Background(), 1, []ChatMessage{{Role: "assistant", Content: "hi"}}); err == nil {
		t.Fatal("expected missing user message error")
	}
	if _, err := agent.Process(context.Background(), 1, []ChatMessage{{Role: "user", Content: "hi"}}); err == nil || !strings.Contains(err.Error(), "chat with LLM") {
		t.Fatalf("client error = %v", err)
	}
}
