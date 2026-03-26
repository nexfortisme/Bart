package shared

import "time"

type Guild struct {
	GuildId   string
	Name      string
	IconUrl   *string
	CreatedAt time.Time
	SyncedAt  time.Time
}

type Channel struct {
	ChannelId string
	GuildId   string
	Name      string
	Type      int
	Topic     *string
	ParentId  *string
	CreatedAt time.Time
	SyncedAt  time.Time
}

type DiscordUser struct {
	DiscordUserId      string
	Username           string
	Discriminator      string
	IsBot              bool
	DataUsageConsent   bool
	DataUsageConsentAt time.Time
	CreatedAt          time.Time
	SyncedAt           time.Time
}

type Member struct {
	GuildId  string
	UserId   string
	Nickname *string
	JoinedAt *time.Time
	SyncedAt time.Time
}

type DiscordMessage struct {
	MessageId string
	ChannelId string
	GuildId   string
	UserId    string
	Content   string
	Type      int
	ReplyToId *string
	ThreadId  *string
	EditedAt  *time.Time
	DeletedAt *time.Time
	CreatedAt time.Time
	SyncedAt  time.Time
}

type MessageIntentClassification struct {
	MessageId      string
	Classification string
	CreatedAt      time.Time
	SyncedAt       time.Time
}

type ToolIntentClassification struct {
	MessageId      string
	Classification string
	CreatedAt      time.Time
	SyncedAt       time.Time
}

type MessageFeedback struct {
	ResponseMessageId string
	OriginalMessageId string
	UserId            string
	Feedback          string
	CreatedAt         time.Time
	SyncedAt          time.Time
}
