package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)


func connectMCP(ctx context.Context) error {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "bart-tools",
		Version: "0.0.1",
	}, nil)

	transport := &mcp.StreamableClientTransport{Endpoint: os.Getenv("MCP_URL")}
	var err error
	mcpSession, err = client.Connect(ctx, transport, nil)
	return err
}

func fetchTools(ctx context.Context) ([]Tool, error) {
	resp, err := mcpSession.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	tools := make([]Tool, len(resp.Tools))
	for i, t := range resp.Tools {

		// fmt.Printf("\nTool Name: %+v\n", t.Name)

		// t.InputSchema is type any (JSON schema as a generic map) — marshal it
		// so the OpenAI API receives the schema object it expects.
		schemaBytes, err := json.Marshal(t.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schema for tool %s: %w", t.Name, err)
		}

		// OpenAI-compatible APIs require "properties" to always be present.
		// The MCP SDK may omit it for tools with no inputs.
		var schemaMap map[string]any
		if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil || schemaMap == nil {
			schemaMap = map[string]any{"type": "object", "properties": map[string]any{}}
		} else if _, ok := schemaMap["properties"]; !ok {
			schemaMap["properties"] = map[string]any{}
		}
		schemaBytes, _ = json.Marshal(schemaMap)

		tools[i].Type = "function"
		tools[i].Function.Name = t.Name
		tools[i].Function.Description = t.Description
		tools[i].Function.Parameters = json.RawMessage(schemaBytes)
	}
	return tools, nil
}

func callTool(ctx context.Context, name string, argsJSON string) (string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid tool arguments: %w", err)
	}

	fmt.Printf("\nCalling Tool: %s with args: %+v\n", name, args)

	result, err := mcpSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}

	for _, c := range result.Content {
		if text, ok := c.(*mcp.TextContent); ok {
			return text.Text, nil
		}
	}
	return "", nil
}
