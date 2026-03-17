package shared

import "time"

func GetDiscordUser(discordUserId string) (*DiscordUser, error) {
	user := &DiscordUser{}
	// RunQuery expects output to be a pointer to a struct/slice/basic type.
	// `user` is already a *DiscordUser, so we should pass `user`, not `&user`.
	err := RunQuery(`SELECT * FROM DiscordUsers WHERE DiscordUserId = ?`, user, discordUserId)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func CreateDiscordUser(discordUserId string, username string, discriminator string, isBot bool) error {
	return UpsertDiscordUser(&DiscordUser{
		DiscordUserId: discordUserId,
		Username: username,
		Discriminator: discriminator,
		IsBot: isBot,
		DataUsageConsent: false,
		CreatedAt: time.Now(),
		SyncedAt: time.Now(),
	})
}

func UpsertDiscordUser(user *DiscordUser) error {
	query := `
	INSERT INTO DiscordUsers (DiscordUserId, Username, Discriminator, IsBot, DataUsageConsent, DataUsageConsentAt, CreatedAt, SyncedAt)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(DiscordUserId) DO UPDATE SET
		Username = ?,
		Discriminator = ?,
		IsBot = ?,
		DataUsageConsent = ?,
		DataUsageConsentAt = ?,
		SyncedAt = CURRENT_TIMESTAMP
	RETURNING *
	`

	// Note: the SQL has 8 placeholders in VALUES(...) plus 5 placeholders in the UPDATE SET.
	// Keep the argument order aligned with the placeholders.
	err := RunQuery(
		query,
		user,
		user.DiscordUserId,
		user.Username,
		user.Discriminator,
		user.IsBot,
		user.DataUsageConsent,
		user.DataUsageConsentAt,
		user.CreatedAt,
		user.SyncedAt,
		// UPDATE SET bindings (same fields as above)
		user.Username,
		user.Discriminator,
		user.IsBot,
		user.DataUsageConsent,
		user.DataUsageConsentAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func UpdateDiscordUserDataUsageConsent(discordUserId string, dataUsageConsent bool) error {

	user, err := GetDiscordUser(discordUserId)
	if err != nil {
		return err
	}

	user.DataUsageConsent = dataUsageConsent
	user.DataUsageConsentAt = time.Now()

	return UpsertDiscordUser(user)
}

func DiscordUserConsents(discordUserId string) (bool, error) {
	user, err := GetDiscordUser(discordUserId)
	if err != nil || user == nil { // Default to false if the user is not found or there is an error
		return false, nil
	}
	return user.DataUsageConsent, nil
}