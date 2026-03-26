package bot

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/nexfortisme/bart/internal/shared"
)

const (
	ThumbsUpReaction   = "👍"
	ThumbsDownReaction = "👎"
)

type PromptState struct {
	AllowedUsers      map[string]struct{}         // set of user IDs
	Reference         *discordgo.MessageReference // original message reference
	OriginalMessageID string
}

var (
	pendingRepliesMu sync.RWMutex
	pendingReplies   = make(map[string]PromptState) // replied message ID -> allowed user
)

func onReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	fmt.Printf("Reaction added: %s to message %s\n", r.Emoji.Name, r.MessageID)

	// Ignore the bot's own reaction events.
	if r.UserID == s.State.User.ID {
		return
	}

	pendingRepliesMu.RLock()
	state, exists := pendingReplies[r.MessageID]
	pendingRepliesMu.RUnlock()

	if !exists {
		fmt.Printf("No prompt state found for message %s\n", r.MessageID)
		return
	}

	// Check if user is in the set
	if _, allowed := state.AllowedUsers[r.UserID]; !allowed {
		fmt.Printf("User %s is not allowed to react to message %s", r.UserID, r.MessageID)
		return
	}

	switch r.Emoji.Name {
	case ThumbsUpReaction:
		// // _, err := s.ChannelMessageSend(r.ChannelID, "<@"+r.UserID+"> selected thumbs up.")
		// _, err := s.ChannelMessageSendReply(r.ChannelID, "<@"+r.UserID+"> selected thumbs up.", state.Reference)
		// if err != nil {
		// 	fmt.Printf("failed to send thumbs up response: %v", err)
		// }

		fmt.Printf("User %s selected thumbs up for message %s\n", r.UserID, r.MessageID)
		if err := shared.UpsertMessageFeedback(r.MessageID, state.OriginalMessageID, r.UserID, "Positive"); err != nil {
			fmt.Printf("failed to save positive feedback for message %s: %v\n", r.MessageID, err)
			return
		}

		sendConfirmationMessage(s, r.ChannelID, r.MessageID)

		// Done with this prompt
		pendingRepliesMu.Lock()
		delete(pendingReplies, r.MessageID)
		fmt.Printf("Deleted prompt state for message %s\n", r.MessageID)
		pendingRepliesMu.Unlock()

	case ThumbsDownReaction:
		// _, err := s.ChannelMessageSend(r.ChannelID, "<@"+r.UserID+"> selected thumbs down.")
		// if err != nil {
		// 	fmt.Printf("failed to send thumbs down response: %v", err)
		// }

		fmt.Printf("User %s selected thumbs down for message %s\n", r.UserID, r.MessageID)
		if err := shared.UpsertMessageFeedback(r.MessageID, state.OriginalMessageID, r.UserID, "Negative"); err != nil {
			fmt.Printf("failed to save negative feedback for message %s: %v\n", r.MessageID, err)
			return
		}

		sendConfirmationMessage(s, r.ChannelID, r.MessageID)

		// Done with this prompt
		pendingRepliesMu.Lock()
		delete(pendingReplies, r.MessageID)
		fmt.Printf("Deleted prompt state for message %s\n", r.MessageID)
		pendingRepliesMu.Unlock()
	}
}

func sendConfirmationMessage(s *discordgo.Session, channelId string, messageId string) {

	originalMessage, err := s.ChannelMessage(channelId, messageId)
	if err != nil {
		fmt.Printf("failed to get original message: %v", err)
		return
	}

	s.ChannelMessageSendReply(originalMessage.ChannelID, "Feedback Recieved. Thank you!", originalMessage.Reference())
}

func AddPendingReply(messageID string, userIDs []string, ref *discordgo.MessageReference, originalMessageID string) {
	set := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		set[id] = struct{}{}
	}

	pendingRepliesMu.Lock()
	pendingReplies[messageID] = PromptState{
		AllowedUsers:      set,
		Reference:         ref,
		OriginalMessageID: originalMessageID,
	}
	pendingRepliesMu.Unlock()
}

func RemovePendingReply(messageID string) {
	pendingRepliesMu.Lock()
	delete(pendingReplies, messageID)
	pendingRepliesMu.Unlock()
}
