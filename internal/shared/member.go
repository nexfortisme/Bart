package shared

func GetMember(guildId, userId string) (*Member, error) {
	member := &Member{}
	err := RunQuery(`SELECT * FROM Members WHERE GuildId = ? AND UserId = ?`, &member, guildId, userId)
	if err != nil {
		return nil, err
	}
	return member, nil
}

func UpsertMember(member *Member) error {
	query := `
	INSERT INTO Members (GuildId, UserId, Nickname, JoinedAt, SyncedAt)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(GuildId, UserId) DO UPDATE SET
		Nickname = ?,
		JoinedAt = ?,
		SyncedAt = CURRENT_TIMESTAMP
	RETURNING *
	`

	err := RunQuery(query, &member, member.GuildId, member.UserId, member.Nickname, member.JoinedAt)
	if err != nil {
		return err
	}

	return nil
}