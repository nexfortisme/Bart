package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func fetcherMCPEndpoint() string {
	if ep := os.Getenv("FETCHER_MCP_ENDPOINT"); ep != "" {
		return ep
	}
	return "http://localhost:3000/mcp"
}

func callFetcherTool(toolName string, args map[string]any) (string, error) {
	ctx := context.Background()

	client := mcp.NewClient(&mcp.Implementation{Name: "bart", Version: "0.0.1"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:             fetcherMCPEndpoint(),
		DisableStandaloneSSE: true,
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return "", fmt.Errorf("failed to connect to fetcher MCP at %s: %w", fetcherMCPEndpoint(), err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("%s failed: %w", toolName, err)
	}
	if result.IsError {
		return "", fmt.Errorf("%s returned error", toolName)
	}

	var sb strings.Builder
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}

	if sb.Len() > 0 {
		return sb.String(), nil
	}

	if result.StructuredContent != nil {
		encoded, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return "", fmt.Errorf("failed to marshal %s structured content: %w", toolName, err)
		}
		return string(encoded), nil
	}

	return "", nil
}

// FetchURL proxies the fetch_url tool exposed by fetcher-mcp.
func FetchURL(args map[string]any) (string, error) {
	pageURL, ok := args["url"].(string)
	if !ok || pageURL == "" {
		return "", fmt.Errorf("url argument required")
	}

	return callFetcherTool("fetch_url", args)
}

// FetchURLs proxies the fetch_urls tool exposed by fetcher-mcp.
func FetchURLs(args map[string]any) (string, error) {
	urls, ok := args["urls"].([]string)
	if ok {
		if len(urls) == 0 {
			return "", fmt.Errorf("urls argument required")
		}
		return callFetcherTool("fetch_urls", args)
	}

	rawURLs, ok := args["urls"].([]any)
	if !ok || len(rawURLs) == 0 {
		return "", fmt.Errorf("urls argument required")
	}

	return callFetcherTool("fetch_urls", args)
}
