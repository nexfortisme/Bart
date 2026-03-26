package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func chat(ctx context.Context, userMessage string, classifiedTool string) (string, error) {
	tools, err := fetchTools(ctx)
	if err != nil {
		fmt.Printf("Warning: could not fetch tools from MCP: %v — continuing without tools", err)
		tools = nil
	}

	messages := []Message{
		{Role: "system", Content: fetchSystemPrompt()},
		{Role: "user", Content: userMessage},
	}

	for {
		resp, err := chatCompletion(messages, tools)
		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("empty response from LLM")
		}

		choice := resp.Choices[0]
		messages = append(messages, choice.Message)

		// No tool calls — model gave us a final answer
		if choice.FinishReason != "tool_calls" || len(choice.Message.ToolCalls) == 0 {
			if content, ok := choice.Message.Content.(string); ok {
				return content, nil
			}
			return "", fmt.Errorf("unexpected content type in response")
		}

		// Execute each tool call via MCP and feed results back
		for _, tc := range choice.Message.ToolCalls {
			result, err := callTool(ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				result = fmt.Sprintf(`{"error": "%s"}`, err.Error())
			}

			messages = append(messages, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
		// Loop: send updated conversation history back to the model
	}
}

func chatCompletion(messages []Message, tools []Tool) (*ChatResponse, error) {
	req := ChatRequest{
		Model:    os.Getenv("LLM_MODEL"),
		Messages: messages,
		Tools:    tools,
	}
	if len(tools) > 0 {
		req.ToolChoice = "auto"
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", os.Getenv("LLM_BASE_URL")+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// httpReq.Header.Set("Authorization", "Bearer "+llmAPIKey) // Don't need an API key for local LM Studio

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}
	return &chatResp, nil
}

func fetchSystemPrompt() string {
	systemPrompt, err := os.ReadFile("./resources/prompts/system_prompt.md") // Relative to main.go
	if err != nil {
		return ""
	}
	return string(systemPrompt)
}
