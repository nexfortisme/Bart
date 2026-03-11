package mcp

import (
    "context"
    "log"
    "net/http"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/nexfortisme/bart/internal/tools"
)

// Typed input structs — the SDK generates JSON schema from these automatically
type WeatherInput struct {
    City string `json:"city" jsonschema:"the city to get weather for"`
}

type SearchInput struct {
    Query string `json:"query" jsonschema:"the search query"`
}

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

// func searchHandler(ctx context.Context, req *mcp.CallToolRequest, in SearchInput) (*mcp.CallToolResult, any, error) {
//     result, err := tools.SearchDocs(in.Query)
//     if err != nil {
//         return nil, nil, err
//     }
//     return &mcp.CallToolResult{
//         Content: []mcp.Content{&mcp.TextContent{Text: result}},
//     }, nil, nil
// }

func Start(addr string) error {
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "bart-tools",
        Version: "0.0.1",
    }, nil)

    // Register tools — schema generated automatically from WeatherInput/SearchInput
    mcp.AddTool(server, &mcp.Tool{
        Name:        "get_weather",
        Description: "Get current weather for a city",
    }, weatherHandler)

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})
	
    // SDK handles the HTTP transport, discovery, routing, and JSON-RPC
    http.Handle("/mcp", handler)
    log.Println("MCP Server Started On: " + addr)
    return http.ListenAndServe(addr, nil)
}