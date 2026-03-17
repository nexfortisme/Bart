package shared

func UpsertMessage(message *DiscordMessage) error {
	query := `
	INSERT INTO Messages (MessageId, ChannelId, GuildId, UserId, Content, Type, ReplyToId, ThreadId, EditedAt, DeletedAt, CreatedAt, SyncedAt)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(MessageId) DO UPDATE SET
		ChannelId = ?,
		GuildId = ?,
		UserId = ?,
		Content = ?,
		Type = ?,
		ReplyToId = ?,
		ThreadId = ?,
		EditedAt = ?,
		DeletedAt = ?,
		SyncedAt = CURRENT_TIMESTAMP
	RETURNING *
	`

	err := RunQuery(query, &message, message.MessageId, message.ChannelId, message.GuildId, message.UserId, message.Content, message.Type, message.ReplyToId, message.ThreadId, message.EditedAt, message.DeletedAt)
	if err != nil {
		return err
	}

	return nil
}

func GetReplyToMessage(messageId string) (*DiscordMessage, error) {
	message := &DiscordMessage{}
	err := RunQuery(`SELECT * FROM Messages WHERE ReplyToId = ?`, &message, messageId)
	if err != nil {
		return nil, err
	}
	return message, nil
}