package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Tool interface {
	ToolInput() any
	ToolHandlerFn() func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error)
	GetTool() *mcp.Tool
}