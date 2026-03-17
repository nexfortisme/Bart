package shared

func GetDiscordUser(discordUserId string) (*DiscordUser, error) {
	user := &DiscordUser{}
	err := RunQuery(`SELECT * FROM DiscordUsers WHERE DiscordUserId = ?`, &user, discordUserId)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func UpsertDiscordUser(user *DiscordUser) error {
	query := `
	INSERT INTO DiscordUsers (DiscordUserId, Username, DisplayName, Discriminator, AvatarUrl, IsBot, CreatedAt, SyncedAt)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(DiscordUserId) DO UPDATE SET
		Username = ?,
		DisplayName = ?,
		Discriminator = ?,
		AvatarUrl = ?,
		IsBot = ?,
		SyncedAt = CURRENT_TIMESTAMP
	RETURNING *
	`

	err := RunQuery(query, &user, user.DiscordUserId, user.Username, user.DisplayName, user.Discriminator, user.AvatarUrl, user.IsBot)
	if err != nil {
		return err
	}

	return nil
}