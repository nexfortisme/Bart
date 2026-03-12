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
		DisplayName     TEXT,
		Discriminator   TEXT,
		AvatarUrl       TEXT,
		IsBot           BOOLEAN NOT NULL DEFAULT 0,
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
		Type        INTEGER NOT NULL DEFAULT 0,
		ReplyToId   TEXT REFERENCES DiscordMessages(MessageId),
		ThreadId    TEXT,
		EditedAt    DATETIME,
		DeletedAt   DATETIME,
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

	createDiscordMessagesDeletedIndex := `
	CREATE INDEX IF NOT EXISTS idx_discord_messages_deleted
	ON DiscordMessages(DeletedAt) WHERE DeletedAt IS NOT NULL;`

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
		createDiscordMessagesDeletedIndex,
	}

	for _, table := range tables {
		err := sqlitex.Execute(db, table, nil)
		if err != nil {
			log.Fatalf("Error creating table: %v", err)
		}
	}

	fmt.Println("Database tables initialized successfully.")
}