package bot

import (
	"context"
	"fmt"
	"strings"
)

// Chat exposes the LLM for use outside the bot package (e.g. CLI chat mode).
// It connects to MCP if a session isn't already open, and strips the dev-mode
// invoke string from the message if the user accidentally included it.
func (b *Bot) Chat(ctx context.Context, message string) (string, error) {
	if b.DevModeInvokeString != "" {
		message = strings.TrimPrefix(message, b.DevModeInvokeString)
		message = strings.TrimSpace(message)
	}

	if mcpSession == nil {
		if err := connectMCP(ctx); err != nil {
			fmt.Printf("Warning: could not connect to MCP: %v\n", err)
		}
	}

	return chat(ctx, message, "")
}
