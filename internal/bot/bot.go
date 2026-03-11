package bot

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	bot            *Bot
	discordSession *discordgo.Session
)

type Bot struct {
	DiscordToken   string
	DiscordSession *discordgo.Session
}

func NewBot(discordToken string) *Bot {
	return &Bot{DiscordToken: discordToken}
}

func (b *Bot) Start() {
	var err error
	b.DiscordSession, err = discordgo.New("Bot " + b.DiscordToken)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	b.DiscordSession.AddHandler(messageReceive)

	err = b.DiscordSession.Open()
	if err != nil {
		fmt.Println("Error opening Discord session:", err)
		return
	}

	fmt.Println("Bot started")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc

	// b.Stop()
}

func (b *Bot) Stop() {
	_ = b.DiscordSession.Close()
	fmt.Println("Bot stopped")
}

func messageReceive(s *discordgo.Session, m *discordgo.MessageCreate) {
	fmt.Println("Message received:", m.Content)

	if m.Content == "Hi Bart" {
		s.ChannelMessageSend(m.ChannelID, "Fuck You.")
	}
}
