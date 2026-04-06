package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nexfortisme/bart/internal/bot"
	"github.com/nexfortisme/bart/internal/classifier"
	"github.com/nexfortisme/bart/internal/logging"
	internalMCP "github.com/nexfortisme/bart/internal/mcp"
	"github.com/nexfortisme/bart/internal/shared"

	"github.com/joho/godotenv"
)

var (
	fiveMinuteTicker = time.NewTicker(5 * time.Minute)
	interrupt        = make(chan os.Signal, 1)

	discordBot *bot.Bot

	logger *log.Logger
	loggerCleanupFunc func() error 

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

	// -- Logging --
	logger, loggerCleanupFunc, err := logging.InitLogging()
	if err != nil {
		fmt.Printf("Error initializing logging: %v", err)
		return
	}
	defer loggerCleanupFunc()
	logger.Println("Logging initialized")

	// -- Seeding Embeddings --
	// One off operation to be completed separate from normal operation
	if seedEmbeddings {
		logger.Println("Seeding embeddings into the database...")
		classifier.SeedEmbeddingsDataset() // TODO - Might make sense to have the path be a command line argument
		logger.Println("Embeddings seeded into the database")
		return
	}

	// -- Discord Bot --
	discordBot = bot.NewBot(os.Getenv("DISCORD_TOKEN"), logger)

	// -- Database --
	dbPool := shared.GetDB()
	defer dbPool.Close()

	// -- Dev Mode --
	if devMode {
		devModeInvokeString = randomString(12)
		discordBot.SetDevModeInvokeString(devModeInvokeString)
		logger.Printf("Dev mode enabled, invoke string: [%s]\n", devModeInvokeString)
	}

	// -- Start Goroutines --
	go discordBot.Start()
	go internalMCP.Start(os.Getenv("MCP_SERVER_ADDRESS"))

	// -- Signal Handling --
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	for {
		select {
			case <- fiveMinuteTicker.C:
				// logger.Println("Five minute ticker")
			case <- interrupt:
				logger.Println("Interrupt signal received")
				fiveMinuteTicker.Stop()
				discordBot.Stop()
				logger.Println("Stopping goroutines")
				return
		}
	}
}

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}
