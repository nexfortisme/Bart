package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nexfortisme/bart/internal/bot"
	"github.com/nexfortisme/bart/internal/classifier"
	internalMCP "github.com/nexfortisme/bart/internal/mcp"
	"github.com/nexfortisme/bart/internal/shared"

	"github.com/joho/godotenv"
)

var (
	fiveMinuteTicker = time.NewTicker(5 * time.Minute)
	interrupt        = make(chan os.Signal, 1)

	discordBot *bot.Bot

	seedEmbeddings bool
)

// Mostly for loading the .env file
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

	// -- Variables Set --
	fmt.Println("\nSecrets Set:")
	fmt.Println("Discord Token Set: \t", os.Getenv("DISCORD_TOKEN") != "", "\n")

	fmt.Println("Models Set:")
	fmt.Println("LLM Model Set: \t\t", os.Getenv("LLM_MODEL") != "")
	fmt.Println("Embeddings Model Set: \t", os.Getenv("EMBEDDING_MODEL") != "", "\n")

	fmt.Println("Base URLs Set:")
	fmt.Println("LLM Base URL Set: \t", os.Getenv("LLM_BASE_URL") != "")
	fmt.Println("MCP Server Address Set: ", os.Getenv("MCP_SERVER_ADDRESS") != "")
	fmt.Println("MCP URL Set: \t\t", os.Getenv("MCP_URL") != "")
	fmt.Println("--------------------------------\n")
}

func main() {

	// -- Command Line Arguments --
	flag.BoolVar(&seedEmbeddings, "seed", false, "Seed embeddings into the database")
	flag.Parse()

	// -- Seeding Embeddings --
	// One off operation to be completed separate from normal operation
	if seedEmbeddings {
		fmt.Println("Seeding embeddings into the database...")
		classifier.SeedEmbeddingsDataset()
		fmt.Println("Embeddings seeded into the database")
		return
	}

	discordBot = bot.NewBot(os.Getenv("DISCORD_TOKEN"))

	dbPool := shared.GetDB()
	defer dbPool.Close()

	go discordBot.Start()
	go internalMCP.Start(os.Getenv("MCP_SERVER_ADDRESS"))

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
			return
		}
	}
}
