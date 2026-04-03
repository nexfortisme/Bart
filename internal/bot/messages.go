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
	"github.com/nexfortisme/bart/internal/shared"
)

var (
	mcpSession *mcp.ClientSession
)

func MessageReceive(stores map[string]*classifier.MemoryStore, devModeInvokeString string) func(s *discordgo.Session, m *discordgo.MessageCreate) {
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

		// printMessageDebugInformation(m)
		// s.ChannelTyping(m.ChannelID)

		messageIntentResult := MessageIntendedForBartClassifier(m.Content, stores)
		fmt.Println("Message Intent Result:", messageIntentResult)

		toolResult := ToolIntentClassifier(m.Content, stores)
		fmt.Println("Tool Result:", toolResult)

		userConsents, err := shared.DiscordUserConsents(m.Author.ID)
		if err != nil {
			fmt.Println("Error getting user consents:", err)
			return
		}

		// If User Consents, Store Message and Classificaiton
		if userConsents {
			err = shared.UpsertMessage(m)
			if err != nil {
				fmt.Println("Error upserting message:", err)
				return
			}

			err = shared.UpsertMessageIntentClassification(m.ID, messageIntentResult)
			if err != nil {
				fmt.Println("Error upserting message intent classification:", err)
				return
			}

			err = shared.UpsertToolIntentClassification(m.ID, toolResult)
			if err != nil {
				fmt.Println("Error upserting tool intent classification:", err)
				return
			}
		}

		if messageIntentResult == classifier.MessageIntentAmbient {
			fmt.Println("Skipping reply: ambient message intent")
			return
		}

		if devModeInvokeString != "" && !strings.HasPrefix(m.Content, devModeInvokeString) {
			return
		}
		userText := strings.TrimSpace(strings.TrimPrefix(m.Content, devModeInvokeString))
		if userText == "" {
			return
		}

		fmt.Println("Connecting to MCP")
		err = connectMCP(context.Background())
		if err != nil {
			fmt.Printf("Error connecting to MCP: %v", err)
			return
		}

		s.ChannelTyping(m.ChannelID)
		fmt.Printf("Message from %s: %s\n", m.Author.Username, userText)

		transcript, err := buildChatTranscript(s, m, userConsents, userText)
		if err != nil {
			fmt.Printf("Error building chat transcript: %v", err)
			s.ChannelMessageSend(m.ChannelID, "Sorry, I ran into an error processing that.")
			return
		}

		response, err := chat(context.Background(), transcript, messageIntentResult, toolResult)
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
		addReactionsToResponse(s, responseMessage, m.Message)

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

	AddPendingReply(m.ID, []string{originalMessage.Author.ID}, m.Reference(), originalMessage.ID)
}

func channelIDForMessageReference(ref *discordgo.MessageReference, fallback string) string {
	if ref != nil && ref.ChannelID != "" {
		return ref.ChannelID
	}
	return fallback
}

// buildChatTranscript maps Discord context into LLM messages. With consent, the full reply chain
// is included; without consent, only the single referenced message (if any) plus the current text.
func buildChatTranscript(s *discordgo.Session, m *discordgo.MessageCreate, userConsents bool, currentUserText string) ([]Message, error) {
	botID := s.State.User.ID
	channelID := m.ChannelID

	if userConsents {
		return transcriptFromReplyChain(s, channelID, m.Message, botID, currentUserText)
	}

	if m.MessageReference != nil && m.MessageReference.MessageID != "" {
		refCh := channelIDForMessageReference(m.MessageReference, channelID)
		prior, err := s.ChannelMessage(refCh, m.MessageReference.MessageID)
		if err != nil {
			return []Message{{Role: "user", Content: currentUserText}}, nil
		}
		priorText := strings.TrimSpace(prior.Content)
		combined := fmt.Sprintf("Previous message:\n%s\n\nCurrent message:\n%s", priorText, currentUserText)
		return []Message{{Role: "user", Content: combined}}, nil
	}

	return []Message{{Role: "user", Content: currentUserText}}, nil
}

func transcriptFromReplyChain(s *discordgo.Session, channelID string, leaf *discordgo.Message, botUserID, currentUserText string) ([]Message, error) {
	const maxDepth = 50
	var chain []*discordgo.Message
	cur := leaf
	for cur != nil && len(chain) < maxDepth {
		chain = append(chain, cur)
		if cur.MessageReference == nil || cur.MessageReference.MessageID == "" {
			break
		}
		refCh := channelIDForMessageReference(cur.MessageReference, channelID)
		parent, err := s.ChannelMessage(refCh, cur.MessageReference.MessageID)
		if err != nil {
			break
		}
		cur = parent
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	var out []Message
	for i, msg := range chain {
		content := strings.TrimSpace(msg.Content)
		if i == len(chain)-1 {
			content = currentUserText
		}
		if content == "" {
			continue
		}
		if msg.Author != nil && msg.Author.Bot && msg.Author.ID != botUserID {
			continue
		}
		role := "user"
		if msg.Author != nil && msg.Author.ID == botUserID {
			role = "assistant"
		}
		out = append(out, Message{Role: role, Content: content})
	}
	if len(out) == 0 {
		return []Message{{Role: "user", Content: currentUserText}}, nil
	}
	return out, nil
}

func printMessageDebugInformation(m *discordgo.MessageCreate) {
	fmt.Println("Message Debug Information (Content): ", m.Content)
	fmt.Println("Message Debug Information (Author ID): ", m.Author.ID)
	fmt.Println("Message Debug Information (Author Username): ", m.Author.Username)
	fmt.Println("Message Debug Information (Author Discriminator): ", m.Author.Discriminator)
	fmt.Println("Message Debug Information (Author Bot): ", m.Author.Bot)
	fmt.Println("Message Debug Information (Author Avatar URL): ", m.Author.AvatarURL(""))
	fmt.Println("Message Debug Information (Message ID): ", m.ID)
	fmt.Println("Message Debug Information (Reference Message ID): ", m.Reference().MessageID)
	fmt.Println("Message Debug Information (Reference Channel ID): ", m.Reference().ChannelID)
	fmt.Println("Message Debug Information (Reference Guild ID): ", m.Reference().GuildID)
	fmt.Println("Message Debug Information (Channel ID): ", m.ChannelID)
	fmt.Println("Message Debug Information (Guild ID): ", m.GuildID)
	fmt.Println("Message Debug Information (Timestamp): ", m.Timestamp)
	fmt.Println("Message Debug Information (Edited Timestamp): ", m.EditedTimestamp)
	fmt.Println("Message Debug Information (TTS): ", m.TTS)
	fmt.Println("Message Debug Information (Embeds): ", m.Embeds)
	fmt.Println("Message Debug Information (Attachments): ", m.Attachments)
}
