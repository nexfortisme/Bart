package bot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/nexfortisme/bart/internal/classifier"
)

func mcpToolNamesForIntent(ti classifier.ToolIntent) []string {
	switch ti {
	case classifier.ToolIntentWeather:
		return []string{"get_weather"}
	case classifier.ToolIntentTime:
		return []string{"get_time"}
	case classifier.ToolIntentWebSearch:
		return []string{"web_search", "fetch_url", "fetch_urls"}
	case classifier.ToolIntentWebFetch:
		return []string{"fetch_url", "fetch_urls"}
	default:
		return nil
	}
}

func toolsForRequest(mi classifier.MessageIntent, ti classifier.ToolIntent, all []Tool) []Tool {
	if mi == classifier.MessageIntentAmbiguous {
		return all
	}
	names := mcpToolNamesForIntent(ti)
	if len(names) == 0 {
		return nil
	}
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[n] = struct{}{}
	}
	var out []Tool
	for _, t := range all {
		if _, ok := want[t.Function.Name]; ok {
			out = append(out, t)
		}
	}
	return out
}

func reasoningEffortFor(mi classifier.MessageIntent, ti classifier.ToolIntent) string {
	if mi == classifier.MessageIntentDirected && ti == classifier.ToolIntentNull {
		return "none"
	}
	return ""
}

// chat runs a multi-turn LLM session. transcript is user/assistant messages only (no system);
// the system prompt is prepended internally.
func chat(ctx context.Context, transcript []Message, messageIntent classifier.MessageIntent, toolIntent classifier.ToolIntent) (string, error) {
	allTools, err := fetchTools(ctx)
	if err != nil {
		fmt.Printf("Warning: could not fetch tools from MCP: %v — continuing without tools", err)
		allTools = nil
	}

	tools := toolsForRequest(messageIntent, toolIntent, allTools)
	reasoning := reasoningEffortFor(messageIntent, toolIntent)

	messages := append([]Message{{Role: "system", Content: fetchSystemPrompt()}}, transcript...)

	for {
		choice, err := chatCompletionStream(ctx, messages, tools, reasoning)
		if err != nil {
			return "", err
		}

		messages = append(messages, choice.Message)

		if choice.FinishReason != "tool_calls" || len(choice.Message.ToolCalls) == 0 {
			if content, ok := choice.Message.Content.(string); ok {
				return content, nil
			}
			return "", fmt.Errorf("unexpected content type in response")
		}

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
	}
}

type streamToolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type streamChoiceChunk struct {
	Delta struct {
		Content   *string             `json:"content"`
		ToolCalls []streamToolCallDelta `json:"tool_calls"`
	} `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

type streamEnvelope struct {
	Choices []streamChoiceChunk `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type accumulatedToolCall struct {
	id        string
	name      string
	arguments string
}

func chatCompletionStream(ctx context.Context, messages []Message, tools []Tool, reasoningEffort string) (*struct {
	Message      Message
	FinishReason string
}, error) {
	req := ChatRequest{
		Model:           os.Getenv("LLM_MODEL"),
		Messages:        messages,
		Tools:           tools,
		Stream:          true,
		ReasoningEffort: reasoningEffort,
	}
	if len(tools) > 0 {
		req.ToolChoice = "auto"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", os.Getenv("LLM_BASE_URL")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM error %d: %s", resp.StatusCode, string(respBody))
	}

	var contentBuf strings.Builder
	toolAcc := make(map[int]*accumulatedToolCall)
	var finishReason string

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read LLM stream: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}

		var env streamEnvelope
		if err := json.Unmarshal([]byte(payload), &env); err != nil {
			continue
		}
		if env.Error != nil && env.Error.Message != "" {
			return nil, fmt.Errorf("LLM stream error: %s", env.Error.Message)
		}
		if len(env.Choices) == 0 {
			continue
		}
		ch := env.Choices[0]
		if ch.Delta.Content != nil {
			contentBuf.WriteString(*ch.Delta.Content)
		}
		for _, d := range ch.Delta.ToolCalls {
			tc := toolAcc[d.Index]
			if tc == nil {
				tc = &accumulatedToolCall{}
				toolAcc[d.Index] = tc
			}
			if d.ID != "" {
				tc.id = d.ID
			}
			if d.Function.Name != "" {
				tc.name = d.Function.Name
			}
			tc.arguments += d.Function.Arguments
		}
		if ch.FinishReason != nil && *ch.FinishReason != "" {
			finishReason = *ch.FinishReason
		}
	}

	indices := make([]int, 0, len(toolAcc))
	for idx := range toolAcc {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	var toolCalls []ToolCall
	for _, idx := range indices {
		tc := toolAcc[idx]
		if tc.id == "" || tc.name == "" {
			continue
		}
		toolCalls = append(toolCalls, ToolCall{
			ID:   tc.id,
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{Name: tc.name, Arguments: tc.arguments},
		})
	}

	assistantMsg := Message{
		Role:      "assistant",
		Content:   contentBuf.String(),
		ToolCalls: toolCalls,
	}

	if finishReason == "" {
		if len(toolCalls) > 0 {
			finishReason = "tool_calls"
		} else {
			finishReason = "stop"
		}
	}

	return &struct {
		Message      Message
		FinishReason string
	}{Message: assistantMsg, FinishReason: finishReason}, nil
}

func fetchSystemPrompt() string {
	systemPrompt, err := os.ReadFile("./resources/prompts/system_prompt.md") // Relative to main.go
	if err != nil {
		return ""
	}
	return string(systemPrompt)
}
