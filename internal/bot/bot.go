package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/nexfortisme/bart/internal/bot/commands"
	"github.com/nexfortisme/bart/internal/classifier"
)

var (
	storePath = "resources/classifier/store.json"

	slashCommands = []*discordgo.ApplicationCommand{
		(&commands.Consent{}).ApplicationCommand(),
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"consent": (&commands.Consent{}).Execute,
	}
)

type Bot struct {
	DiscordToken        string
	DiscordSession      *discordgo.Session
	ClassifierStore     *classifier.MemoryStore
	DevModeInvokeString string
}

func NewBot(discordToken string) *Bot {
	return &Bot{DiscordToken: discordToken, DevModeInvokeString: ""}
}

func (b *Bot) SetDevModeInvokeString(invokeString string) {
	b.DevModeInvokeString = invokeString
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

	b.ClassifierStore = classifier.NewStore()
	b.ClassifierStore.Load(storePath)

	b.DiscordSession.AddHandler(MessageReceive(b.ClassifierStore, b.DevModeInvokeString))
	b.DiscordSession.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuilds)

	err = b.DiscordSession.Open()
	if err != nil {
		fmt.Println("Error opening Discord session:", err)
		return
	}

	registerSlashCommands(b.DiscordSession)

	fmt.Println("Bot started")
}

func (b *Bot) Stop() {
	_ = b.DiscordSession.Close()
	fmt.Println("Bot stopped")
}

func registerSlashCommands(s *discordgo.Session) {
	fmt.Println("Registering Commands...")
	// Used for adding slash commands
	// Add the command and then add the handler for that command
	// https://github.com/bwmarrin/discordgo/blob/master/examples/slash_commands/main.go
	registeredCommands := make([]*discordgo.ApplicationCommand, len(slashCommands))
	for i, v := range slashCommands {
		fmt.Printf("Registering command: %v\n", v.Name)
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			fmt.Printf("Cannot create '%v' command: %v\n", v.Name, err)
		}
		registeredCommands[i] = cmd
	}
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
			// break
		// case discordgo.InteractionMessageComponent:
		// 	commandNameSplit := strings.Split(i.MessageComponentData().CustomID, ":")
		// 	if h, ok := applicationCommandHandlers[commandNameSplit[0]]; ok {
		// 		h(s, i)
		// 	}
		// 	break
		default:
			fmt.Printf("Unknown interaction type: %v\n", i.Type)
		}

	})
}

func removeRegisteredSlashCommands(s *discordgo.Session) {
	fmt.Println("Removing Commands...")

	registeredCommands, err := s.ApplicationCommands(s.State.User.ID, "")
	if err != nil {
		fmt.Printf("Could not fetch registered commands: %v\n", err)
	}

	for _, v := range registeredCommands {
		fmt.Printf("Removing command: %v\n", v.Name)
		err := s.ApplicationCommandDelete(s.State.User.ID, "", v.ID)
		if err != nil {
			fmt.Printf("Cannot delete '%v' command: %v\n", v.Name, err)
		}
	}
}
