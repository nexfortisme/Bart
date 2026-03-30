package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nexfortisme/bart/internal/bot"
	"github.com/nexfortisme/bart/internal/classifier"
	"github.com/nexfortisme/bart/internal/cli"
	internalMCP "github.com/nexfortisme/bart/internal/mcp"
	"github.com/nexfortisme/bart/internal/shared"

	"github.com/joho/godotenv"
)

var (
	fiveMinuteTicker = time.NewTicker(5 * time.Minute)
	interrupt        = make(chan os.Signal, 1)

	discordBot *bot.Bot

	devModeInvokeString = ""
	characters          = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	seedEmbeddings bool
	devMode        bool
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
	fmt.Printf("Discord Token Set: \t %t\n\n", os.Getenv("DISCORD_TOKEN") != "")

	fmt.Println("Models Set:")
	fmt.Println("LLM Model Set: \t\t", os.Getenv("LLM_MODEL") != "")
	fmt.Printf("Embeddings Model Set: \t %t\n\n", os.Getenv("EMBEDDING_MODEL") != "")

	fmt.Println("Base URLs Set:")
	fmt.Println("LLM Base URL Set: \t", os.Getenv("LLM_BASE_URL") != "")
	fmt.Println("MCP Server Address Set: ", os.Getenv("MCP_SERVER_ADDRESS") != "")
	fmt.Println("MCP URL Set: \t\t", os.Getenv("MCP_URL") != "")
	fmt.Printf("--------------------------------\n")
}

func main() {

	// -- Command Line Arguments --
	flag.BoolVar(&seedEmbeddings, "seed", false, "Seed embeddings into the database")
	flag.BoolVar(&devMode, "dev", false, "Run in development mode")
	flag.Parse()

	// -- Seeding Embeddings --
	// One off operation to be completed separate from normal operation
	if seedEmbeddings {
		fmt.Println("Seeding embeddings into the database...")
		classifier.SeedEmbeddingsDataset() // TODO - Might make sense to have the path be a command line argument
		fmt.Println("Embeddings seeded into the database")
		return
	}

	discordBot = bot.NewBot(os.Getenv("DISCORD_TOKEN"))

	if devMode {
		devModeInvokeString = randomString(12)
		fmt.Printf("\nDev mode enabled, invoke string: [%s]\n\n", devModeInvokeString)
		discordBot.SetDevModeInvokeString(devModeInvokeString)
	}

	dbPool := shared.GetDB()
	defer dbPool.Close()

	// Capture stdout before starting goroutines so bot operational logs
	// are buffered and only shown when the user selects "Watch logs".
	lm := cli.Capture()

	go discordBot.Start()
	go internalMCP.Start(os.Getenv("MCP_SERVER_ADDRESS"))

	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	cli.NewMenu(discordBot, lm, interrupt).Run()

	fmt.Fprint(lm.RealOut, "\033[2K") // Clear the current line
	fmt.Fprint(lm.RealOut, "\033[0G") // Move cursor to the beginning of the line
	fmt.Fprintln(lm.RealOut, "Stopping...")
	fiveMinuteTicker.Stop()
	discordBot.Stop()
}

func randomString(length int) string {
	// rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}
