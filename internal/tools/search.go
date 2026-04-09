package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool Interface Implementation
type SearchTool struct {
	Tool
}

type SearchInput struct {
	Query string `json:"query" jsonschema:"the search query"`
}

func (t *SearchTool) ToolInput() any {
	return SearchInput{}
}

func (t *SearchTool) ToolHandlerFn() func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
		input, ok := in.(SearchInput)
		if !ok {
			return nil, nil, fmt.Errorf("invalid input type for web_search")
		}
		result, err := webSearch(map[string]any{"query": input.Query})
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil, nil
	}
}

func (t *SearchTool) GetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "web_search",
		Description: "Search the web using SearXNG and return a list of results with titles, URLs, and snippets",
		// InputSchema: t.ToolInput(),
	}
}

// Tool Utility Implementation
func searxngBaseURL() string {
	if ep := os.Getenv("SEARXNG_URL"); ep != "" {
		return ep
	}
	return "http://localhost:8888"
}

type searxngResponse struct {
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
		Engine  string `json:"engine"`
	} `json:"results"`
}

func webSearch(args map[string]any) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query argument required")
	}

	searchURL := fmt.Sprintf("%s/search?q=%s&format=json", searxngBaseURL(), url.QueryEscape(query))
	resp, err := http.Get(searchURL)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("searxng returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var raw searxngResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	out, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}
	return string(out), nil
}
