package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func MessageReceive(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignoring messages from self
	if m.Author.ID == s.State.User.ID {
		fmt.Println("Ignoring message from self")
		return
	}
	if m.Author.Bot {
		fmt.Println("Ignoring message from bot")
		return
	}

	fmt.Printf("Message received: %+v\n", m)
	fmt.Printf("Message Content: %v\n", m.Message.Content)

	if m.Message.Content == "Hi Bart" {
		s.ChannelMessageSend(m.ChannelID, "Fuck You.")
	}
}
