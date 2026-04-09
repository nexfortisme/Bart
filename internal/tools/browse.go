package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool Interface Implementation
type FetchURLTool struct {
	Tool
}

type FetchURLsTool struct {
	Tool
}

type FetchOptions struct {
	Timeout           *int    `json:"timeout,omitempty" jsonschema:"Page loading timeout in milliseconds, default is 30000"`
	WaitUntil         *string `json:"waitUntil,omitempty" jsonschema:"When navigation is considered complete: load, domcontentloaded, networkidle, or commit. Default is load"`
	ExtractContent    *bool   `json:"extractContent,omitempty" jsonschema:"Whether to intelligently extract the main content. Default is true"`
	MaxLength         *int    `json:"maxLength,omitempty" jsonschema:"Maximum length of returned content in characters"`
	ReturnHTML        *bool   `json:"returnHtml,omitempty" jsonschema:"Whether to return HTML instead of Markdown. Default is false"`
	WaitForNavigation *bool   `json:"waitForNavigation,omitempty" jsonschema:"Whether to wait for additional navigation after the initial page load. Default is false"`
	NavigationTimeout *int    `json:"navigationTimeout,omitempty" jsonschema:"Maximum wait for additional navigation in milliseconds. Default is 10000"`
	DisableMedia      *bool   `json:"disableMedia,omitempty" jsonschema:"Whether to disable images, stylesheets, fonts, and media. Default is true"`
	Debug             *bool   `json:"debug,omitempty" jsonschema:"Whether to enable debug mode and show the browser window"`
}

type FetchURLInput struct {
	URL string `json:"url" jsonschema:"The URL of the web page to fetch"`
	FetchOptions
}

type FetchURLsInput struct {
	URLs []string `json:"urls" jsonschema:"Array of URLs to fetch in parallel"`
	FetchOptions
}

func (t *FetchURLTool) ToolInput() any {
	return FetchURLInput{}
}

func (t *FetchURLsTool) ToolInput() any {
	return FetchURLsInput{}
}

func (t *FetchURLTool) ToolHandlerFn() func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {

		input, ok := in.(FetchURLInput)
		if !ok {
			return nil, nil, fmt.Errorf("invalid input type for fetch_url")
		}

		args := fetchArgsFromOptions(input.FetchOptions)
		args["url"] = input.URL

		result, err := fetchURL(args)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil, nil
	}
}

func (t *FetchURLsTool) ToolHandlerFn() func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
		input, ok := in.(FetchURLsInput)
		if !ok {
			return nil, nil, fmt.Errorf("invalid input type for fetch_urls")
		}

		args := fetchArgsFromOptions(input.FetchOptions)
		args["urls"] = input.URLs

		result, err := fetchURLs(args)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil, nil
	}
}

func (t *FetchURLTool) GetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "fetch_url",
		Description: "Retrieve web page content from a URL using fetcher-mcp. Supports JavaScript rendering, Markdown extraction, HTML output, navigation waits, and fetch tuning options.",
		// InputSchema: t.ToolInput(),
	}
}

func (t *FetchURLsTool) GetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "fetch_urls",
		Description: "Batch retrieve web page content from multiple URLs in parallel using fetcher-mcp. Accepts the same fetch options as fetch_url.",
		// InputSchema: t.ToolInput(),
	}
}

func fetcherMCPEndpoint() string {
	if ep := os.Getenv("FETCHER_MCP_ENDPOINT"); ep != "" {
		return ep
	}
	return "http://localhost:3000/mcp"
}

// FetchURL proxies the fetch_url tool exposed by fetcher-mcp.
func fetchURL(args map[string]any) (string, error) {
	pageURL, ok := args["url"].(string)
	if !ok || pageURL == "" {
		return "", fmt.Errorf("url argument required")
	}

	return callFetcherTool("fetch_url", args)
}

// FetchURLs proxies the fetch_urls tool exposed by fetcher-mcp.
func fetchURLs(args map[string]any) (string, error) {
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

func fetchArgsFromOptions(opts FetchOptions) map[string]any {
	args := map[string]any{}

	if opts.Timeout != nil {
		args["timeout"] = *opts.Timeout
	}
	if opts.WaitUntil != nil {
		args["waitUntil"] = *opts.WaitUntil
	}
	if opts.ExtractContent != nil {
		args["extractContent"] = *opts.ExtractContent
	}
	if opts.MaxLength != nil {
		args["maxLength"] = *opts.MaxLength
	}
	if opts.ReturnHTML != nil {
		args["returnHtml"] = *opts.ReturnHTML
	}
	if opts.WaitForNavigation != nil {
		args["waitForNavigation"] = *opts.WaitForNavigation
	}
	if opts.NavigationTimeout != nil {
		args["navigationTimeout"] = *opts.NavigationTimeout
	}
	if opts.DisableMedia != nil {
		args["disableMedia"] = *opts.DisableMedia
	}
	if opts.Debug != nil {
		args["debug"] = *opts.Debug
	}

	return args
}
