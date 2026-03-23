package mcp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nexfortisme/bart/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Typed input structs — the SDK generates JSON schema from these automatically
type WeatherInput struct {
	City string `json:"city" jsonschema:"the city to get weather for"`
}

type SearchInput struct {
	Query string `json:"query" jsonschema:"the search query"`
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

type CurrentTimeInput struct{}

// Tool handlers — strongly typed, no map[string]any parsing
func weatherHandler(ctx context.Context, req *mcp.CallToolRequest, in WeatherInput) (*mcp.CallToolResult, any, error) {
	result, err := tools.GetWeather(map[string]any{"city": in.City})
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

func searchHandler(ctx context.Context, req *mcp.CallToolRequest, in SearchInput) (*mcp.CallToolResult, any, error) {
	result, err := tools.WebSearch(map[string]any{"query": in.Query})
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
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

func fetchURLHandler(ctx context.Context, req *mcp.CallToolRequest, in FetchURLInput) (*mcp.CallToolResult, any, error) {
	args := fetchArgsFromOptions(in.FetchOptions)
	args["url"] = in.URL

	result, err := tools.FetchURL(args)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

func fetchURLsHandler(ctx context.Context, req *mcp.CallToolRequest, in FetchURLsInput) (*mcp.CallToolResult, any, error) {
	args := fetchArgsFromOptions(in.FetchOptions)
	args["urls"] = in.URLs

	result, err := tools.FetchURLs(args)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

func currentTimeHandler(ctx context.Context, req *mcp.CallToolRequest, in CurrentTimeInput) (*mcp.CallToolResult, any, error) {
	result, err := tools.GetTime(nil)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

func Start(addr string) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "bart-tools",
		Version: "0.0.1",
	}, nil)

	// Register tools — schema generated automatically from input structs
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_weather",
		Description: "Get current weather for a city",
	}, weatherHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "web_search",
		Description: "Search the web using SearXNG and return a list of results with titles, URLs, and snippets",
	}, searchHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fetch_url",
		Description: "Retrieve web page content from a URL using fetcher-mcp. Supports JavaScript rendering, Markdown extraction, HTML output, navigation waits, and fetch tuning options.",
	}, fetchURLHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fetch_urls",
		Description: "Batch retrieve web page content from multiple URLs in parallel using fetcher-mcp. Accepts the same fetch options as fetch_url.",
	}, fetchURLsHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_time",
		Description: "Get the current time",
	}, currentTimeHandler)

	// StreamableHTTPHandler is a handler that streams the response from the server to the client
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})

	// SDK handles the HTTP transport, discovery, routing, and JSON-RPC
	http.Handle("/mcp", handler)
	fmt.Println("MCP Server Started On: " + addr)
	return http.ListenAndServe(addr, nil)
}
