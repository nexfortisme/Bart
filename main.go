package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nexfortisme/bart/internal/bot"
	internalMCP "github.com/nexfortisme/bart/internal/mcp"
	"github.com/nexfortisme/bart/internal/shared"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	fiveMinuteTicker = time.NewTicker(5 * time.Minute)
	interrupt        = make(chan os.Signal, 1)

	discordToken string
	discordBot   *bot.Bot

	mcpUrl     = "http://localhost:8090"
	mcpSession *mcp.ClientSession
)

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working directory: %v", err)
	}

	envFilePath := filepath.Join(cwd, ".env")
	err = godotenv.Overload(envFilePath)
	if err != nil {
		fmt.Printf("Error loading .env file: %v", err)
	}

	discordToken = os.Getenv("DISCORD_TOKEN")
}

func main() {
	discordBot = bot.NewBot(os.Getenv("DISCORD_TOKEN"))

	dbPool := shared.GetDB()
	defer dbPool.Close()

	go discordBot.Start()
	go internalMCP.Start(mcpUrl)

	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// Main Loop
	for {
		select {
		case <-fiveMinuteTicker.C:
		case <-interrupt:
			fmt.Print("\033[2K") // Clear the current line
			fmt.Print("\033[0G") // Move cursor to the beginning of the line
			fmt.Println("Interrupt received, stopping...")
			fiveMinuteTicker.Stop()
			discordBot.Stop()

			if mcpSession != nil {
				mcpSession.Close()
			}
			return
		}
	}
}

func connectMCP(ctx context.Context) error {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "bart-tools",
		Version: "0.0.1",
	}, nil)

	var err error

	transport := &mcp.StreamableClientTransport{Endpoint: mcpUrl}
	mcpSession, err = client.Connect(ctx, transport, nil)
	return err
}
