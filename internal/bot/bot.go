package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"

	"github.com/nexfortisme/bart/internal/bot/commands"
	"github.com/nexfortisme/bart/internal/classifier"
)

var (
	classifierStores = map[string]string{
		"message_intent": classifier.MessageIntentStorePath,
		"tool_intent":    classifier.ToolIntentStorePath,
	}

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
	ClassifierStores    map[string]*classifier.MemoryStore
	DevModeInvokeString string
	Logger              *log.Logger
}

func NewBot(discordToken string, logger *log.Logger) *Bot {
	return &Bot{DiscordToken: discordToken, DevModeInvokeString: "", Logger: logger}
}

func (b *Bot) SetDevModeInvokeString(invokeString string) {
	b.DevModeInvokeString = invokeString
}

func (b *Bot) InDevMode() bool {
	return b.DevModeInvokeString != ""
}

// Invite Link: https://discord.com/api/v9/oauth2/authorize?client_id= <CLIENT_ID> &permissions=517547084864&scope=bot
// Will also need to have Message Content Intent enabled in the bot's settings in the Discord Developer Portal.
func (b *Bot) Start() {
	var err error
	b.DiscordSession, err = discordgo.New("Bot " + b.DiscordToken)
	if err != nil {
		b.Logger.Println("Error creating Discord session:", err)
		return
	}

	b.ClassifierStores = make(map[string]*classifier.MemoryStore)
	for name, path := range classifierStores {
		store := classifier.NewStore()
		store.Load(path)
		b.ClassifierStores[name] = store
	}

	// Handlers for Messages and Reactions
	b.DiscordSession.AddHandler(MessageReceive(b.ClassifierStores, b.DevModeInvokeString, b.Logger))
	b.DiscordSession.AddHandler(onReactionAdd)

	// Intent Flags for Bot Operation
	b.DiscordSession.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent | discordgo.IntentsGuilds | discordgo.IntentGuildMessageReactions)

	err = b.DiscordSession.Open()
	if err != nil {
		b.Logger.Println("Error opening Discord session:", err)
		return
	}

	registerSlashCommands(b.DiscordSession, b.Logger)

	b.Logger.Println("Bot started")
}

func (b *Bot) Stop() {
	_ = b.DiscordSession.Close()
	b.Logger.Println("Bot stopped")
}

func registerSlashCommands(s *discordgo.Session, logger *log.Logger) {
	logger.Println("Registering Commands...")
	// Used for adding slash commands
	// Add the command and then add the handler for that command
	// https://github.com/bwmarrin/discordgo/blob/master/examples/slash_commands/main.go
	registeredCommands := make([]*discordgo.ApplicationCommand, len(slashCommands))
	for i, v := range slashCommands {
		logger.Printf("Registering command: %v\n", v.Name)
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			logger.Printf("Cannot create '%v' command: %v\n", v.Name, err)
		}
		registeredCommands[i] = cmd
	}
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		default:
			logger.Printf("Unknown interaction type: %v\n", i.Type)
		}

	})
}

func removeRegisteredSlashCommands(s *discordgo.Session, logger *log.Logger) {
	logger.Println("Removing Commands...")

	registeredCommands, err := s.ApplicationCommands(s.State.User.ID, "")
	if err != nil {
		logger.Printf("Could not fetch registered commands: %v\n", err)
	}

	for _, v := range registeredCommands {
		logger.Printf("Removing command: %v\n", v.Name)
		err := s.ApplicationCommandDelete(s.State.User.ID, "", v.ID)
		if err != nil {
			logger.Printf("Cannot delete '%v' command: %v\n", v.Name, err)
		}
	}
}
