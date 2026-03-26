package shared

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

var (
	dbPool *sqlitex.Pool
	once   sync.Once
)

func initDB() {
	var err error

	// Get database path from environment variable, fallback to "db.sqlite"
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "db.sqlite"
	}

	dbPool, err = sqlitex.NewPool(dbPath, sqlitex.PoolOptions{
		PoolSize: 12,
	})
	if err != nil {
		fmt.Printf("Error connecting to database at %s.\n", dbPath)
		panic(err)
	}

	conn, err := dbPool.Take(context.Background())
	if err != nil {
		panic(err)
	}
	defer dbPool.Put(conn)

	InitializeDatabase(conn)

	fmt.Printf("Database Connected at %s.\n", dbPath)
}

func GetDB() *sqlitex.Pool {
	once.Do(func() {
		initDB()
	})
	return dbPool
}

func GetConn(ctx context.Context) (*sqlite.Conn, error) {
	conn, err := GetDB().Take(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting database connection: %w", err)
	}
	return conn, nil
}

func PutConn(conn *sqlite.Conn) {
	GetDB().Put(conn)
}

func InitializeDatabase(db *sqlite.Conn) {
	createGuildsTable := `
	CREATE TABLE IF NOT EXISTS Guilds (
		GuildId     TEXT PRIMARY KEY,
		Name        TEXT NOT NULL,
		IconUrl     TEXT,
		CreatedAt   DATETIME NOT NULL,
		SyncedAt    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	createChannelsTable := `
	CREATE TABLE IF NOT EXISTS Channels (
		ChannelId   TEXT PRIMARY KEY,
		GuildId     TEXT NOT NULL REFERENCES Guilds(GuildId),
		Name        TEXT NOT NULL,
		Type        INTEGER NOT NULL,
		Topic       TEXT,
		ParentId    TEXT,
		CreatedAt   DATETIME NOT NULL,
		SyncedAt    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	createDiscordUsersTable := `
	CREATE TABLE IF NOT EXISTS DiscordUsers (
		DiscordUserId   TEXT PRIMARY KEY,
		Username        TEXT NOT NULL,
		Discriminator   TEXT,
		IsBot           BOOLEAN NOT NULL DEFAULT 0,
		DataUsageConsent BOOLEAN NOT NULL DEFAULT 0,
		DataUsageConsentAt DATETIME,
		CreatedAt       DATETIME NOT NULL,	
		SyncedAt        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	createDiscordMembersTable := `
	CREATE TABLE IF NOT EXISTS Members (
		GuildId     TEXT NOT NULL REFERENCES Guilds(GuildId),
		UserId      TEXT NOT NULL REFERENCES DiscordUsers(DiscordUserId),
		Nickname    TEXT,
		JoinedAt    DATETIME,
		SyncedAt    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (GuildId, UserId)
	);`

	createDiscordMessagesTable := `
	CREATE TABLE IF NOT EXISTS DiscordMessages (
		MessageId   TEXT PRIMARY KEY,
		ChannelId   TEXT NOT NULL REFERENCES Channels(ChannelId),
		GuildId     TEXT NOT NULL REFERENCES Guilds(GuildId),
		UserId      TEXT NOT NULL REFERENCES DiscordUsers(DiscordUserId),
		Content     TEXT NOT NULL DEFAULT '',
		ReplyToId   TEXT REFERENCES DiscordMessages(MessageId),
		CreatedAt   DATETIME NOT NULL,
		SyncedAt    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	createDiscordMessagesChannelIndex := `
	CREATE INDEX IF NOT EXISTS idx_discord_messages_channel
	ON DiscordMessages(ChannelId, CreatedAt DESC);`

	createDiscordMessagesUserIndex := `
	CREATE INDEX IF NOT EXISTS idx_discord_messages_user
	ON DiscordMessages(UserId, CreatedAt DESC);`

	createDiscordMessagesGuildIndex := `
	CREATE INDEX IF NOT EXISTS idx_discord_messages_guild
	ON DiscordMessages(GuildId, CreatedAt DESC);`

	createDiscordMessagesReplyIndex := `
	CREATE INDEX IF NOT EXISTS idx_discord_messages_reply
	ON DiscordMessages(ReplyToId) WHERE ReplyToId IS NOT NULL;`

	createMessageIntentClassificationsTable := `
	CREATE TABLE IF NOT EXISTS MessageIntentClassifications (
		MessageId       TEXT PRIMARY KEY REFERENCES DiscordMessages(MessageId) ON DELETE CASCADE,
		Classification  TEXT NOT NULL,
		CreatedAt       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		SyncedAt        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	createToolIntentClassificationsTable := `
	CREATE TABLE IF NOT EXISTS ToolIntentClassifications (
		MessageId       TEXT PRIMARY KEY REFERENCES DiscordMessages(MessageId) ON DELETE CASCADE,
		Classification  TEXT NOT NULL,
		CreatedAt       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		SyncedAt        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	createMessageIntentClassificationsIndex := `
	CREATE INDEX IF NOT EXISTS idx_message_intent_classification
	ON MessageIntentClassifications(Classification, CreatedAt DESC);`

	createToolIntentClassificationsIndex := `
	CREATE INDEX IF NOT EXISTS idx_tool_intent_classification
	ON ToolIntentClassifications(Classification, CreatedAt DESC);`

	createMessageFeedbackTable := `
	CREATE TABLE IF NOT EXISTS MessageFeedback (
		ResponseMessageId TEXT NOT NULL,
		OriginalMessageId TEXT NOT NULL,
		UserId            TEXT NOT NULL REFERENCES DiscordUsers(DiscordUserId),
		Feedback          TEXT NOT NULL,
		CreatedAt         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		SyncedAt          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (ResponseMessageId, UserId)
	);`

	createMessageFeedbackResponseIndex := `
	CREATE INDEX IF NOT EXISTS idx_message_feedback_response
	ON MessageFeedback(ResponseMessageId, CreatedAt DESC);`

	createMessageFeedbackOriginalIndex := `
	CREATE INDEX IF NOT EXISTS idx_message_feedback_original
	ON MessageFeedback(OriginalMessageId, CreatedAt DESC);`

	tables := []string{
		createGuildsTable,
		createChannelsTable,
		createDiscordUsersTable,
		createDiscordMembersTable,
		createDiscordMessagesTable,
		createDiscordMessagesChannelIndex,
		createDiscordMessagesUserIndex,
		createDiscordMessagesGuildIndex,
		createDiscordMessagesReplyIndex,
		createMessageIntentClassificationsTable,
		createToolIntentClassificationsTable,
		createMessageIntentClassificationsIndex,
		createToolIntentClassificationsIndex,
		createMessageFeedbackTable,
		createMessageFeedbackResponseIndex,
		createMessageFeedbackOriginalIndex,
	}

	for _, table := range tables {
		err := sqlitex.Execute(db, table, nil)
		if err != nil {
			log.Fatalf("Error creating table: %v", err)
		}
	}

	fmt.Println("Database tables initialized successfully.")
}
