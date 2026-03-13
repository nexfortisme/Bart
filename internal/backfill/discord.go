package backfill

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/nexfortisme/bart/internal/classifier"
	"github.com/nexfortisme/bart/internal/shared"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

const (
	classifierVersion   = "current-store-v1"
	storePath           = "resources/classifier/store.json"
	maxBatchSize        = 100
	classifierThreshold = 0.5
)

type ChannelBackfillOptions struct {
	ChannelID string
	Limit     int
}

type topMatch struct {
	Text       string  `json:"text"`
	Intent     string  `json:"intent"`
	Similarity float32 `json:"similarity"`
}

func BackfillChannel(ctx context.Context, discordToken string, opts ChannelBackfillOptions) error {
	if opts.ChannelID == "" {
		return fmt.Errorf("channel ID is required")
	}

	session, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		return fmt.Errorf("create discord session: %w", err)
	}
	defer session.Close()

	if err := session.Open(); err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}

	channel, err := session.Channel(opts.ChannelID)
	if err != nil {
		return fmt.Errorf("fetch channel %s: %w", opts.ChannelID, err)
	}

	var guild *discordgo.Guild
	if channel.GuildID != "" {
		guild, err = session.Guild(channel.GuildID)
		if err != nil {
			return fmt.Errorf("fetch guild %s: %w", channel.GuildID, err)
		}
	}

	store := classifier.NewStore()
	if err := store.Load(storePath); err != nil {
		return fmt.Errorf("load classifier store: %w", err)
	}

	model := classifier.NewClassifier(
		classifier.NewLMStudioEmbedder(os.Getenv("LLM_BASE_URL"), os.Getenv("EMBEDDING_MODEL")),
		store,
	)

	conn, err := shared.GetConn(ctx)
	if err != nil {
		return err
	}
	defer shared.PutConn(conn)

	if err := upsertChannelMetadata(conn, channel, guild); err != nil {
		return err
	}

	fmt.Printf("Backfilling channel %s (%s)\n", channel.Name, channel.ID)

	remaining := opts.Limit
	beforeID := ""
	totalSaved := 0

	for {
		if remaining == 0 {
			break
		}

		batchSize := maxBatchSize
		if remaining > 0 && remaining < batchSize {
			batchSize = remaining
		}

		messages, err := session.ChannelMessages(channel.ID, batchSize, beforeID, "", "")
		if err != nil {
			return fmt.Errorf("fetch channel messages: %w", err)
		}
		if len(messages) == 0 {
			break
		}

		endTx, err := sqlitex.ImmediateTransaction(conn)
		if err != nil {
			return fmt.Errorf("start sqlite transaction: %w", err)
		}

		func() {
			defer endTx(&err)

			for i := len(messages) - 1; i >= 0; i-- {
				msg := messages[i]
				if err = persistBacktestMessage(conn, msg, model); err != nil {
					return
				}
				totalSaved++
			}
		}()
		if err != nil {
			return err
		}

		beforeID = messages[len(messages)-1].ID
		fmt.Printf("Saved %d messages so far; oldest message in batch: %s\n", totalSaved, beforeID)

		if remaining > 0 {
			remaining -= len(messages)
			if remaining <= 0 {
				break
			}
		}

		if len(messages) < batchSize {
			break
		}
	}

	fmt.Printf("Backfill complete. Saved %d messages into SQLite.\n", totalSaved)
	return nil
}

func upsertChannelMetadata(conn *sqlite.Conn, channel *discordgo.Channel, guild *discordgo.Guild) error {
	if guild != nil {
		guildCreatedAt := time.Now().UTC()
		if ts, err := discordgo.SnowflakeTimestamp(guild.ID); err == nil {
			guildCreatedAt = ts.UTC()
		}

		if err := sqlitex.Execute(conn, `
			INSERT INTO Guilds (GuildId, Name, IconUrl, CreatedAt)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(GuildId) DO UPDATE SET
				Name = excluded.Name,
				IconUrl = excluded.IconUrl,
				SyncedAt = CURRENT_TIMESTAMP
		`, &sqlitex.ExecOptions{
			Args: []any{
				guild.ID,
				guild.Name,
				emptyToNil(guild.Icon),
				guildCreatedAt,
			},
		}); err != nil {
			return fmt.Errorf("upsert guild: %w", err)
		}
	}

	var createdAt any
	if channel.ID != "" {
		if ts, err := discordgo.SnowflakeTimestamp(channel.ID); err == nil {
			createdAt = ts
		}
	}
	if createdAt == nil {
		createdAt = time.Now().UTC()
	}

	if err := sqlitex.Execute(conn, `
		INSERT INTO Channels (ChannelId, GuildId, Name, Type, Topic, ParentId, CreatedAt)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(ChannelId) DO UPDATE SET
			GuildId = excluded.GuildId,
			Name = excluded.Name,
			Type = excluded.Type,
			Topic = excluded.Topic,
			ParentId = excluded.ParentId,
			SyncedAt = CURRENT_TIMESTAMP
	`, &sqlitex.ExecOptions{
		Args: []any{
			channel.ID,
			emptyToNil(channel.GuildID),
			channel.Name,
			int(channel.Type),
			emptyToNil(channel.Topic),
			emptyToNil(channel.ParentID),
			createdAt,
		},
	}); err != nil {
		return fmt.Errorf("upsert channel: %w", err)
	}

	return nil
}

func persistBacktestMessage(conn *sqlite.Conn, msg *discordgo.Message, model *classifier.Classifier) error {
	result, err := model.Classify(msg.Content)
	if err != nil {
		return fmt.Errorf("classify message %s: %w", msg.ID, err)
	}

	topMatches, err := marshalTopMatches(result.TopMatches)
	if err != nil {
		return fmt.Errorf("marshal top matches for %s: %w", msg.ID, err)
	}

	var threadID any
	if msg.Thread != nil {
		threadID = msg.Thread.ID
	}

	authorIsBot := false
	if msg.Author != nil {
		authorIsBot = msg.Author.Bot
	}

	if err := sqlitex.Execute(conn, `
		INSERT INTO ClassifierBacktestMessages (
			MessageId,
			ChannelId,
			GuildId,
			ThreadId,
			Content,
			AuthorIsBot,
			MessageType,
			ClassifierIntent,
			ClassifierConfidence,
			ClassifierTopMatches,
			CreatedAt,
			ClassifierVersion,
			ClassifierThreshold
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(MessageId) DO UPDATE SET
			ChannelId = excluded.ChannelId,
			GuildId = excluded.GuildId,
			ThreadId = excluded.ThreadId,
			Content = excluded.Content,
			AuthorIsBot = excluded.AuthorIsBot,
			MessageType = excluded.MessageType,
			ClassifierIntent = excluded.ClassifierIntent,
			ClassifierConfidence = excluded.ClassifierConfidence,
			ClassifierTopMatches = excluded.ClassifierTopMatches,
			CreatedAt = excluded.CreatedAt,
			ClassifierVersion = excluded.ClassifierVersion,
			ClassifierThreshold = excluded.ClassifierThreshold,
			ClassifiedAt = CURRENT_TIMESTAMP,
			SyncedAt = CURRENT_TIMESTAMP
	`, &sqlitex.ExecOptions{
		Args: []any{
			msg.ID,
			msg.ChannelID,
			emptyToNil(msg.GuildID),
			threadID,
			msg.Content,
			authorIsBot,
			int(msg.Type),
			result.Intent,
			result.Confidence,
			topMatches,
			msg.Timestamp.UTC(),
			classifierVersion,
			classifierThreshold,
		},
	}); err != nil {
		return fmt.Errorf("persist backtest message %s: %w", msg.ID, err)
	}

	return nil
}

func marshalTopMatches(matches []classifier.QueryResult) (string, error) {
	payload := make([]topMatch, 0, len(matches))
	for _, match := range matches {
		payload = append(payload, topMatch{
			Text:       match.Entry.Text,
			Intent:     match.Entry.Intent,
			Similarity: match.Similarity,
		})
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func emptyToNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}
