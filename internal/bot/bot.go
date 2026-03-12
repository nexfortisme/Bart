package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	DiscordToken   string
	DiscordSession *discordgo.Session
}

func NewBot(discordToken string) *Bot {
	return &Bot{DiscordToken: discordToken}
}

// Invite Link: https://discord.com/api/v9/oauth2/authorize?client_id= <CLIENT_ID> &permissions=517547084864&scope=bot
// Will also need to have Message Content Intent enabled in the bot's settings in the Discord Developer Portal.
func (b *Bot) Start() {
	var err error
	b.DiscordSession, err = discordgo.New("Bot " + b.DiscordToken)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	b.DiscordSession.AddHandler(MessageReceive)
	b.DiscordSession.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuilds)

	err = b.DiscordSession.Open()
	if err != nil {
		fmt.Println("Error opening Discord session:", err)
		return
	}

	fmt.Println("Bot started")
}

func (b *Bot) Stop() {
	_ = b.DiscordSession.Close()
	fmt.Println("Bot stopped")
}
