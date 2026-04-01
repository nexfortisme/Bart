package shared

import (
	"github.com/bwmarrin/discordgo"
	"github.com/nexfortisme/bart/internal/classifier"
)

func UpsertMessage(message *discordgo.MessageCreate) error {
	query := `
	INSERT INTO DiscordMessages (MessageId, ChannelId, GuildId, UserId, Content, ReplyToId, CreatedAt)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(MessageId) DO UPDATE SET
		ChannelId = ?,
		GuildId = ?,
		UserId = ?,
		Content = ?,
		ReplyToId = ?,
		SyncedAt = CURRENT_TIMESTAMP
	`

	err := RunQuery(
		query,
		nil,
		message.ID,
		message.ChannelID,
		message.GuildID,
		message.Author.ID,
		message.Content,
		message.Reference().MessageID,
		message.Timestamp,
		message.ChannelID,
		message.GuildID,
		message.Author.ID,
		message.Content,
		message.Reference().MessageID,
	)
	if err != nil {
		return err
	}

	return nil
}

func GetReplyToMessage(messageId string) (*DiscordMessage, error) {
	message := &DiscordMessage{}
	err := RunQuery(`SELECT * FROM DiscordMessages WHERE ReplyToId = ?`, &message, messageId)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func UpsertMessageIntentClassification(messageId string, classification classifier.MessageIntent) error {
	query := `
	INSERT INTO MessageIntentClassifications (MessageId, Classification)
	VALUES (?, ?)
	ON CONFLICT(MessageId) DO UPDATE SET
		Classification = ?,
		SyncedAt = CURRENT_TIMESTAMP
	`

	return RunQuery(query, nil, messageId, string(classification), string(classification))
}

func UpsertToolIntentClassification(messageId string, classification classifier.ToolIntent) error {
	query := `
	INSERT INTO ToolIntentClassifications (MessageId, Classification)
	VALUES (?, ?)
	ON CONFLICT(MessageId) DO UPDATE SET
		Classification = ?,
		SyncedAt = CURRENT_TIMESTAMP
	`

	return RunQuery(query, nil, messageId, string(classification), string(classification))
}

func UpsertMessageFeedback(responseMessageId string, originalMessageId string, userId string, feedback string) error {
	query := `
	INSERT INTO MessageFeedback (ResponseMessageId, OriginalMessageId, UserId, Feedback)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(ResponseMessageId, UserId) DO UPDATE SET
		OriginalMessageId = ?,
		Feedback = ?,
		SyncedAt = CURRENT_TIMESTAMP
	`

	return RunQuery(
		query,
		nil,
		responseMessageId,
		originalMessageId,
		userId,
		feedback,
		originalMessageId,
		feedback,
	)
}
