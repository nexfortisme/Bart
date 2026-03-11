package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nexfortisme/bart/internal/bot"
	"github.com/nexfortisme/bart/internal/mcp"
	
	"github.com/joho/godotenv"
)

var (
	fiveMinuteTicker = time.NewTicker(5 * time.Minute)

	discordToken string
	discordBot   *bot.Bot

	mcpUrl = "localhost:8090"
)

func main() {
	fmt.Println("Hello, World!")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	discordBot = bot.NewBot(os.Getenv("DISCORD_TOKEN"))
	discordBot.Start()

	go mcp.Start(mcpUrl)

	// This is a simple 5 minute loop originally used to save the bot statistics
	for {
		select {
		case <-fiveMinuteTicker.C:
		case <-interrupt:
			fmt.Println("Interrupt received, stopping...")
			fiveMinuteTicker.Stop()
			discordBot.Stop()
			return
		}
	}
}

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
