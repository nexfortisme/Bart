package shared

func GetChannel(channelId string) (*Channel, error) {
	channel := &Channel{}
	err := RunQuery(`SELECT * FROM Channels WHERE ChannelId = ?`, &channel, channelId)
	if err != nil {
		return nil, err
	}
	return channel, nil
}

func UpsertChannel(channel *Channel) error {
	query := `
	INSERT INTO Channels (ChannelId, GuildId, Name, Type, Topic, ParentId, CreatedAt, SyncedAt)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(ChannelId) DO UPDATE SET
		GuildId = ?,
		Name = ?,
		Type = ?,
		Topic = ?,
		ParentId = ?,
		SyncedAt = CURRENT_TIMESTAMP
	RETURNING *
	`

	err := RunQuery(query, &channel, channel.ChannelId, channel.GuildId, channel.Name, channel.Type, channel.Topic, channel.ParentId)
	if err != nil {
		return err
	}

	return nil
}