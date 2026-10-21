package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

const maxToolCallRounds = 3

type ToolExecutor interface {
	Execute(
		ctx context.Context,
		userID int64,
		toolName string,
		arguments string,
		userMessage string,
	) (string, error)
}

type Agent struct {
	client      *Client
	toolHandler ToolExecutor
}

func NewAgent(client *Client, toolHandler ToolExecutor) *Agent {
	return &Agent{
		client:      client,
		toolHandler: toolHandler,
	}
}

func (a *Agent) Process(
	ctx context.Context,
	userID int64,
	messages []ChatMessage,
) (ChatMessage, error) {
	if userID <= 0 {
		return ChatMessage{}, fmt.Errorf("invalid user ID")
	}

	if len(messages) == 0 {
		return ChatMessage{}, fmt.Errorf("messages must not be empty")
	}

	userMessage := lastUserMessage(messages)
	if userMessage == "" {
		return ChatMessage{}, fmt.Errorf("messages must contain a user message")
	}

	// Добавляем системный промпт в начало истории.
	if messages[0].Role != "system" {
		messages = append(
			[]ChatMessage{{
				Role:    "system",
				Content: systemPrompt,
			}},
			messages...,
		)
	}

	// Сохраняем результаты инструментов, чтобы не выполнять
	// повторно один и тот же вызов с теми же аргументами.
	executedCalls := make(map[string]string)

	for round := 1; round <= maxToolCallRounds; round++ {
		log.Printf(
			"LLM tool round: %d/%d",
			round,
			maxToolCallRounds,
		)

		response, err := a.client.ChatWithTools(ctx, messages)
		if err != nil {
			return ChatMessage{}, fmt.Errorf(
				"chat with LLM: %w",
				err,
			)
		}

		log.Printf(
			"LLM response received: tool_calls=%d",
			len(response.ToolCalls),
		)

		if len(response.ToolCalls) == 0 {
			return response, nil
		}

		// Сохраняем сообщение ассистента с вызовами инструментов.
		messages = append(messages, response)

		for _, call := range response.ToolCalls {
			toolName := call.Function.Name
			arguments := call.Function.Arguments

			log.Printf(
				"executing AI tool: name=%s",
				toolName,
			)

			callKey := toolName + "\x00" + arguments

			result, exists := executedCalls[callKey]
			if exists {
				log.Printf(
					"repeated AI tool call: name=%s; reusing result",
					toolName,
				)
			} else {
				result, err = a.toolHandler.Execute(
					ctx,
					userID,
					toolName,
					arguments,
					userMessage,
				)

				if err != nil {
					log.Printf(
						"AI tool execution failed: name=%s, error=%v",
						toolName,
						err,
					)

					errorResult, marshalErr := json.Marshal(
						map[string]string{
							"error": "operation_failed",
							"message": "Операция не выполнена. " +
								"Сообщи пользователю об ошибке.",
						},
					)
					if marshalErr != nil {
						return ChatMessage{}, fmt.Errorf(
							"encode tool error result: %w",
							marshalErr,
						)
					}

					result = string(errorResult)
				}

				executedCalls[callKey] = result
			}

			log.Printf(
				"AI tool execution completed: name=%s",
				toolName,
			)

			messages = append(messages, ChatMessage{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	// Если лимит вызовов исчерпан, генерируем ответ без новых инструментов.
	response, err := a.client.Chat(ctx, messages)
	if err != nil {
		return ChatMessage{}, fmt.Errorf(
			"generate final response after tool calls: %w",
			err,
		)
	}

	if response.Content == "" {
		return ChatMessage{}, fmt.Errorf(
			"LLM returned an empty final response after tool calls",
		)
	}

	return response, nil
}

func lastUserMessage(messages []ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}

	return ""
}
