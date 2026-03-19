package bot

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nexfortisme/bart/internal/classifier"
)

var (
	mcpSession *mcp.ClientSession
)

func MessageReceive(store *classifier.MemoryStore, devModeInvokeString string) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {

		start := time.Now()

		// Ignoring messages from self
		if m.Author.ID == s.State.User.ID {
			fmt.Println("Ignoring message from self")
			return
		}
		if m.Author.Bot {
			fmt.Println("Ignoring message from bot")
			return
		}

		// s.ChannelTyping(m.ChannelID)

		result := MessageIntendedForBartClassifier(m.Content, store)
		fmt.Println("Result:", result)

		// s.ChannelMessageSendReply(m.ChannelID, result, m.Reference())

		// if !result {
		// 	fmt.Println("Message not intended for bot")
		// 	return
		// }

		// If message doesn't start with "test_message", return
		// Just for testing purposes
		if !strings.HasPrefix(m.Content, devModeInvokeString) {
			return
		}

		fmt.Println("Connecting to MCP")
		err := connectMCP(context.Background())
		if err != nil {
			fmt.Printf("Error connecting to MCP: %v", err)
			return
		}

		s.ChannelTyping(m.ChannelID)
		fmt.Printf("Message from %s: %s\n", m.Author.Username, strings.Trim(m.Content, devModeInvokeString))

		response, err := chat(context.Background(), strings.Trim(m.Content, devModeInvokeString))
		if err != nil {
			fmt.Printf("Error: %v", err)
			s.ChannelMessageSend(m.ChannelID, "Sorry, I ran into an error processing that.")
			return
		}

		response = stripThinking(response)

		// Discord has a 2000 character limit per message
		if len(response) > 2000 {
			response = response[:1997] + "..."
		}

		responseMessage, err := s.ChannelMessageSendReply(m.ChannelID, response, m.Reference())
		if err != nil {
			fmt.Printf("Error: %v", err)
			return
		}

		// Adding Reactions to the Response so the User can react to the response
		addReactionsToResponse(s, responseMessage, m.Message);

		duration := time.Since(start)
		fmt.Printf("Handled message from %s in %s\n", m.Author.Username, duration)
	}
}

// StripThinking removes all <think>...</think> blocks from a string.
func stripThinking(input string) string {
	re := regexp.MustCompile(`(?s)<think>.*?</think>`)
	return re.ReplaceAllString(input, "")
}

func addReactionsToResponse(s *discordgo.Session, m *discordgo.Message, originalMessage *discordgo.Message) {
	if err := s.MessageReactionAdd(m.ChannelID, m.ID, ThumbsUpReaction); err != nil {
		fmt.Printf("failed to add thumbs up reaction: %v", err)
	}
	if err := s.MessageReactionAdd(m.ChannelID, m.ID, ThumbsDownReaction); err != nil {
		fmt.Printf("failed to add thumbs down reaction: %v", err)
	}

	AddPendingReply(m.ID, []string{originalMessage.Author.ID}, m.Reference())
}
