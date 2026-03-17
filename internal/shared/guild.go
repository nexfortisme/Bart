package shared

func GetGuild(guildId string) (*Guild, error) {
	guild := &Guild{}
	err := RunQuery(`SELECT * FROM Guilds WHERE GuildId = ?`, &guild, guildId)
	if err != nil {
		return nil, err
	}
	return guild, nil
}

func UpsertGuild(guild *Guild) error {
	query := `
	INSERT INTO Guilds (GuildId, Name, IconUrl, CreatedAt, SyncedAt)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(GuildId) DO UPDATE SET
		Name = ?,
		IconUrl = ?,
		SyncedAt = CURRENT_TIMESTAMP
	RETURNING *
	`

	err := RunQuery(query, &guild, guild.GuildId, guild.Name, guild.IconUrl)
	if err != nil {
		return err
	}

	return nil
}