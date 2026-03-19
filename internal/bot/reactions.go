package bot

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
)

const (
	ThumbsUpReaction = "👍"
	ThumbsDownReaction = "👎"
)

type PromptState struct {
	AllowedUsers map[string]struct{}          // set of user IDs
	Reference    *discordgo.MessageReference  // original message reference
}

var (
	pendingRepliesMu sync.RWMutex
	pendingReplies   = make(map[string]PromptState) // replied message ID -> allowed user
)

func onReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	fmt.Printf("Reaction added: %s to message %s", r.Emoji.Name, r.MessageID)

	// Ignore the bot's own reaction events.
	if r.UserID == s.State.User.ID {
		return
	}

	pendingRepliesMu.RLock()
	state, exists := pendingReplies[r.MessageID]
	pendingRepliesMu.RUnlock()

	if !exists {
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

		fmt.Printf("User %s selected thumbs up for message %s", r.UserID, r.MessageID)
		
		// Done with this prompt
		pendingRepliesMu.Lock()
		delete(pendingReplies, r.MessageID)
		pendingRepliesMu.Unlock()

	case ThumbsDownReaction:
		// _, err := s.ChannelMessageSend(r.ChannelID, "<@"+r.UserID+"> selected thumbs down.")
		// if err != nil {
		// 	fmt.Printf("failed to send thumbs down response: %v", err)
		// }

		fmt.Printf("User %s selected thumbs down for message %s", r.UserID, r.MessageID)

		// Done with this prompt
		pendingRepliesMu.Lock()
		delete(pendingReplies, r.MessageID)
		pendingRepliesMu.Unlock()
	}
}

func AddPendingReply(messageID string, userIDs []string, ref *discordgo.MessageReference) {
	set := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		set[id] = struct{}{}
	}

	pendingRepliesMu.Lock()
	pendingReplies[messageID] = PromptState{
		AllowedUsers: set,
		Reference:    ref,
	}
	pendingRepliesMu.Unlock()
}

func RemovePendingReply(messageID string) {
	pendingRepliesMu.Lock()
	delete(pendingReplies, messageID)
	pendingRepliesMu.Unlock()
}