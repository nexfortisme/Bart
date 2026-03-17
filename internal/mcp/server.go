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

type FetchInput struct {
    URL string `json:"url" jsonschema:"the URL of the web page to fetch"`
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

func fetchHandler(ctx context.Context, req *mcp.CallToolRequest, in FetchInput) (*mcp.CallToolResult, any, error) {
    result, err := tools.FetchPage(map[string]any{"url": in.URL})
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
        Name:        "fetch_page",
        Description: "Fetch a web page via playwright-mcp and return its content as an accessibility snapshot. Requires playwright-mcp running at PLAYWRIGHT_MCP_ENDPOINT (default: http://localhost:3000/mcp)",
    }, fetchHandler)
    
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