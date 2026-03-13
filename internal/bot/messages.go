package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	mcpSession *mcp.ClientSession
)

func MessageReceive(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignoring messages from self
	if m.Author.ID == s.State.User.ID {
		fmt.Println("Ignoring message from self")
		return
	}
	if m.Author.Bot {
		fmt.Println("Ignoring message from bot")
		return
	}

	result := MessageIntendedForBotScored(m.Content)
	if !result {
		fmt.Println("Message not intended for bot")
		return
	}

	fmt.Println("Connecting to MCP")
	err := connectMCP(context.Background())
	if err != nil {
		fmt.Printf("Error connecting to MCP: %v", err)
		return
	}

	s.ChannelTyping(m.ChannelID)
	fmt.Printf("Message from %s: %s", m.Author.Username, m.Content)

	response, err := chat(context.Background(), m.Content)
	if err != nil {
		fmt.Printf("Error: %v", err)
		s.ChannelMessageSend(m.ChannelID, "Sorry, I ran into an error processing that.")
		return
	}

	// Discord has a 2000 character limit per message
	if len(response) > 2000 {
		response = response[:1997] + "..."
	}

	s.ChannelMessageSendReply(m.ChannelID, response, m.Reference())
}

func connectMCP(ctx context.Context) error {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "bart-tools",
		Version: "0.0.1",
	}, nil)

	transport := &mcp.StreamableClientTransport{Endpoint: os.Getenv("MCP_URL")}
	var err error
	mcpSession, err = client.Connect(ctx, transport, nil)
	return err
}

func fetchTools(ctx context.Context) ([]Tool, error) {
	resp, err := mcpSession.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	tools := make([]Tool, len(resp.Tools))
	for i, t := range resp.Tools {

		fmt.Printf("Tool Name: %+v\n", t.Name)

		// t.InputSchema is type any (JSON schema as a generic map) — marshal it
		// so the OpenAI API receives the schema object it expects.
		schemaBytes, err := json.Marshal(t.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schema for tool %s: %w", t.Name, err)
		}
		tools[i].Type = "function"
		tools[i].Function.Name = t.Name
		tools[i].Function.Description = t.Description
		tools[i].Function.Parameters = json.RawMessage(schemaBytes)
	}
	return tools, nil
}

func callTool(ctx context.Context, name string, argsJSON string) (string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid tool arguments: %w", err)
	}

	result, err := mcpSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}

	for _, c := range result.Content {
		if text, ok := c.(*mcp.TextContent); ok {
			return text.Text, nil
		}
	}
	return "", nil
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
	httpReq, err := http.NewRequest("POST", os.Getenv("LLM_BASE_URL") + "/chat/completions", bytes.NewBuffer(body))
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

func chat(ctx context.Context, userMessage string) (string, error) {
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
			fmt.Printf("Tool called: %s → %s", tc.Function.Name, result)

			messages = append(messages, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
		// Loop: send updated conversation history back to the model
	}
}

func fetchSystemPrompt() string {
	// Relative to main.go
	systemPrompt, err := os.ReadFile("./resources/system_prompt.md")
	if err != nil {
		return ""
	}
	return string(systemPrompt)
}
