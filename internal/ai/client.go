package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const maxResponseSize = 2 << 20 // 2 MiB

type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, model string) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)

	if baseURL == "" || apiKey == "" || model == "" {
		return nil, fmt.Errorf("LLM base URL, API key and model must not be empty")
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 180 * time.Second,
		},
	}, nil
}

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolCallFunc `json:"function"`
}

type ToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Tools    []Tool        `json:"tools,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

func (c *Client) Chat(
	ctx context.Context,
	messages []ChatMessage,
) (ChatMessage, error) {
	return c.chat(ctx, messages, nil)
}

func (c *Client) ChatWithTools(
	ctx context.Context,
	messages []ChatMessage,
) (ChatMessage, error) {
	return c.chat(ctx, messages, tools)
}

func (c *Client) chat(
	ctx context.Context,
	messages []ChatMessage,
	availableTools []Tool,
) (ChatMessage, error) {
	requestBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    availableTools,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return ChatMessage{}, fmt.Errorf("marshal LLM request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return ChatMessage{}, fmt.Errorf("create LLM request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ChatMessage{}, fmt.Errorf("send LLM request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		errorBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

		return ChatMessage{}, fmt.Errorf(
			"LLM returned HTTP status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(errorBody)),
		)
	}

	var result chatResponse

	decoder := json.NewDecoder(
		io.LimitReader(resp.Body, maxResponseSize),
	)

	if err := decoder.Decode(&result); err != nil {
		return ChatMessage{}, fmt.Errorf("decode LLM response: %w", err)
	}

	if len(result.Choices) == 0 {
		return ChatMessage{}, fmt.Errorf("LLM returned no choices")
	}

	return result.Choices[0].Message, nil
}
