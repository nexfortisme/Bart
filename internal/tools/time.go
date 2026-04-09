package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool Interface Implementation
type TimeTool struct {
	Tool
}

type TimeInput struct {}

func (t *TimeTool) ToolInput() any {
	return TimeInput{}
}

func (t *TimeTool) ToolHandlerFn() func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in any) (*mcp.CallToolResult, any, error) {
		result, err := getTime(nil)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil, nil
	}
}

func (t *TimeTool) GetTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_time",
		Description: "Get the current time",
		// InputSchema: t.ToolInput(),
	}
}

// Tool Utility Implementation
func getTime(_ map[string]any) (string, error) {
    return fmt.Sprintf(`{"time": "%s"}`, time.Now().Format(time.RFC3339)), nil
}