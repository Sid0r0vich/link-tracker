package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/db"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
)

const (
	configFile   = "./config.yaml"
	chatCount    = 1000
	linksPerChat = 100
	urlPrefix    = "https://example.com"
)

func main() {
	if err := filloutDB(); err != nil {
		panic(err)
	}
}

func filloutDB() error {
	if err := os.Setenv("CONFIG_FILE", configFile); err != nil {
		return err
	}

	logger := logs.NewLogger()
	cfg, err := config.LoadConfig(logger)
	if err != nil {
		return err
	}

	cfg.Database.Host = "localhost"
	connCfg, err := db.GetConnCfg(&cfg.Database)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, connCfg)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `TRUNCATE subscription_tag, chat_subscription, subscription, chat RESTART IDENTITY CASCADE`); err != nil {
		return fmt.Errorf("truncate: %w", err)
	}

	for chatID := range chatCount {
		if _, err := tx.Exec(ctx, `INSERT INTO chat(id) VALUES ($1)`, chatID); err != nil {
			return fmt.Errorf("insert chat: %w", err)
		}

		for linkIdx := range linksPerChat {
			url := fmt.Sprintf("%s/%d/%d", urlPrefix, chatID, linkIdx)

			var subscriptionID int64
			if err := tx.QueryRow(ctx, `INSERT INTO subscription(url, updated_at) VALUES ($1, now()) RETURNING id`, url).Scan(&subscriptionID); err != nil {
				return fmt.Errorf("insert subscription: %w", err)
			}

			var chatSubscriptionID int64
			if err := tx.QueryRow(ctx, `INSERT INTO chat_subscription(chat_id, subscription_id) VALUES ($1, $2) RETURNING id`, chatID, subscriptionID).Scan(&chatSubscriptionID); err != nil {
				return fmt.Errorf("insert chat_subscription: %w", err)
			}

			if _, err := tx.Exec(ctx, `INSERT INTO subscription_tag(chat_subscription_id, tag) VALUES ($1, $2)`, chatSubscriptionID, "tag"); err != nil {
				return fmt.Errorf("insert subscription_tag: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}
