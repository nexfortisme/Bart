package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/nexfortisme/bart/internal/shared"
)

type Consent struct {}

func (c *Consent) ApplicationCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "consent",
		Description: "Consent to usage of your data with the bot",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "consent",
				Description: "Consent to usage of your data with the bot",
				Required:    true,
			},
		},
	}
}

func (c *Consent) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	consent := options[0].Value.(bool)

	err := shared.CreateDiscordUser(i.Interaction.Member.User.ID, i.Interaction.Member.User.Username, i.Interaction.Member.User.Discriminator, i.Interaction.Member.User.Bot)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error creating user: " + err.Error(),
			},
		})
	}

	if consent {

		err = shared.UpdateDiscordUserDataUsageConsent(i.Interaction.Member.User.ID, consent)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Error updating user: " + err.Error(),
				},
			})
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You have consented to usage of your data with the bot. You can remove your consent at any time by using the `/consent` command again. Upon removal of consent, all of your data will be deleted from our systems.",
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	} else {

		err = shared.UpdateDiscordUserDataUsageConsent(i.Interaction.Member.User.ID, consent)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Error updating user: " + err.Error(),
				},
			})
		}

		// TODO - Delete all of the user's data from the database
		// Currently in FAFO territory

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You have NOT consented or REMOVED your consent to usage of your data with the bot. Please use the `/consent` command again if you wish to consent to usage of your data with the bot.",
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	}
}