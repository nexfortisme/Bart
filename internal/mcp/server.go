package mcp

import (
	"fmt"
	"net/http"

	"github.com/nexfortisme/bart/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func Start(addr string) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "bart-tools",
		Version: "0.0.1",
	}, nil)

	// Register tools — schema generated automatically from input structs
	weatherTool := &tools.WeatherTool{}
	mcp.AddTool(server, weatherTool.GetTool(), weatherTool.ToolHandlerFn())

	searchTool := &tools.SearchTool{}
	mcp.AddTool(server, searchTool.GetTool(), searchTool.ToolHandlerFn())

	fetchURLTool := &tools.FetchURLTool{}
	mcp.AddTool(server, fetchURLTool.GetTool(), fetchURLTool.ToolHandlerFn())

	fetchURLsTool := &tools.FetchURLsTool{}
	mcp.AddTool(server, fetchURLsTool.GetTool(), fetchURLsTool.ToolHandlerFn())

	timeTool := &tools.TimeTool{}
	mcp.AddTool(server, timeTool.GetTool(), timeTool.ToolHandlerFn())

	// StreamableHTTPHandler is a handler that streams the response from the server to the client
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{JSONResponse: true})

	// SDK handles the HTTP transport, discovery, routing, and JSON-RPC
	http.Handle("/mcp", handler)
	fmt.Println("MCP Server Started On: " + addr)
	return http.ListenAndServe(addr, nil)
}
