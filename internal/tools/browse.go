package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func playwrightMCPEndpoint() string {
	if ep := os.Getenv("PLAYWRIGHT_MCP_ENDPOINT"); ep != "" {
		return ep
	}
	return "http://localhost:3000/mcp"
}

// FetchPage fetches a web page via playwright-mcp and returns the page content.
// It expects playwright-mcp to be running as an HTTP MCP server (streamable HTTP transport).
// Start it with: npx @playwright/mcp --port 3000
func FetchPage(args map[string]any) (string, error) {
	pageURL, ok := args["url"].(string)
	if !ok || pageURL == "" {
		return "", fmt.Errorf("url argument required")
	}

	ctx := context.Background()

	client := mcp.NewClient(&mcp.Implementation{Name: "bart", Version: "0.0.1"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:             playwrightMCPEndpoint(),
		DisableStandaloneSSE: true,
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return "", fmt.Errorf("failed to connect to playwright-mcp at %s: %w", playwrightMCPEndpoint(), err)
	}
	defer session.Close()

	// Navigate to the URL
	navResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "browser_navigate",
		Arguments: map[string]any{"url": pageURL},
	})
	if err != nil {
		return "", fmt.Errorf("browser_navigate failed: %w", err)
	}
	if navResult.IsError {
		return "", fmt.Errorf("browser_navigate returned error")
	}

	// Get page content as an accessibility snapshot (plain text representation)
	snapResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "browser_snapshot",
		Arguments: map[string]any{},
	})
	if err != nil {
		return "", fmt.Errorf("browser_snapshot failed: %w", err)
	}
	if snapResult.IsError {
		return "", fmt.Errorf("browser_snapshot returned error")
	}

	var sb strings.Builder
	for _, c := range snapResult.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}

	// Close the browser inside playwright-mcp before ending the session.
	// This releases the Chromium process; the playwright-mcp server itself stays running.
	session.CallTool(ctx, &mcp.CallToolParams{ //nolint:errcheck
		Name:      "browser_close",
		Arguments: map[string]any{},
	})

	return sb.String(), nil
}
