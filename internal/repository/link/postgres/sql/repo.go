package sql_link_repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
)

type SqlLinkService struct {
	pool                  *pgxpool.Pool
	subscriptionBatchSize uint
}

func NewSqlLinkService(pool *pgxpool.Pool, subscriptionBatchSize uint) *SqlLinkService {
	return &SqlLinkService{pool: pool, subscriptionBatchSize: subscriptionBatchSize}
}

func (s *SqlLinkService) AddChat(chatID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx, "INSERT INTO chat (id) VALUES ($1)", chatID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return uerrors.ErrChatAlreadyExists
			}
		}

		return fmt.Errorf("insert chat: %w", err)
	}

	return nil
}

func (s *SqlLinkService) DeleteChat(chatID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tag, err := s.pool.Exec(ctx, "DELETE FROM chat WHERE id = $1", chatID)
	if err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return uerrors.ErrChatNotExists
	}

	return nil
}

func (s *SqlLinkService) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, "SELECT * FROM chat WHERE id = $1", chatID)
	if err != nil {
		return nil, fmt.Errorf("query chat: %w", err)
	}

	if !rows.Next() {
		rows.Close()
		return nil, uerrors.ErrChatNotExists
	}
	rows.Close()

	query := `
		SELECT 
			chat_subscription.id, 
			subscription.url, 
			subscription.updated_at, 
			ARRAY_REMOVE(ARRAY_AGG(subscription_tag.tag), NULL) AS tags
		FROM chat_subscription
		JOIN subscription ON chat_subscription.subscription_id = subscription.id
		LEFT JOIN subscription_tag ON chat_subscription.id = subscription_tag.chat_subscription_id 
		WHERE chat_subscription.chat_id = $1
		GROUP BY chat_subscription.id, subscription.url, subscription.updated_at
	`

	rows, err = tx.Query(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("query chat: %w", err)
	}

	var links []domain.LinkWithID
	for rows.Next() {
		var link domain.LinkWithID
		if err := rows.Scan(&link.ID, &link.URL, &link.UpdatedAt, &link.Tags); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		links = append(links, link)
	}
	rows.Close()

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return links, nil
}

func (s *SqlLinkService) AddLink(chatID int64, link domain.Link) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, "SELECT * FROM chat WHERE id = $1", chatID)
	if err != nil {
		return 0, fmt.Errorf("query chat: %w", err)
	}

	if !rows.Next() {
		rows.Close()
		return 0, uerrors.ErrChatNotExists
	}
	rows.Close()

	var subscriptionID int64
	err = tx.QueryRow(ctx, "INSERT INTO subscription (url, updated_at) VALUES ($1, $2) ON CONFLICT (url) DO UPDATE SET url=EXCLUDED.url RETURNING id", link.URL, link.UpdatedAt).Scan(&subscriptionID)
	if err != nil {
		return 0, fmt.Errorf("insert subscription: %w", err)
	}

	var chatSubscriptionID int64
	err = tx.QueryRow(ctx, "INSERT INTO chat_subscription (chat_id, subscription_id) VALUES ($1, $2) RETURNING id", chatID, subscriptionID).Scan(&chatSubscriptionID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return 0, uerrors.ErrLinkAlreadyExists
			}
		}

		return 0, fmt.Errorf("insert chat subscription: %w", err)
	}

	tagsCount := len(link.Tags)
	if tagsCount > 0 {
		params := make([]interface{}, 2*tagsCount)
		valueStrings := make([]string, tagsCount)

		for i, tag := range link.Tags {
			params[2*i] = chatSubscriptionID
			params[2*i+1] = tag
			valueStrings[i] = fmt.Sprintf("($%d, $%d)", 2*i+1, 2*i+2)
		}

		query := fmt.Sprintf(`
    	INSERT INTO subscription_tag (chat_subscription_id, tag) VALUES %s 
    	ON CONFLICT DO NOTHING
	`, strings.Join(valueStrings, ","))

		_, err = tx.Exec(ctx, query, params...)
		if err != nil {
			return 0, fmt.Errorf("insert tags: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return chatSubscriptionID, nil
}

func (s *SqlLinkService) DeleteLink(chatID int64, url string) (*domain.LinkWithID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, "SELECT * FROM chat WHERE id = $1", chatID)
	if err != nil {
		return nil, fmt.Errorf("query chat: %w", err)
	}

	if !rows.Next() {
		rows.Close()
		return nil, uerrors.ErrChatNotExists
	}
	rows.Close()

	query := `
		DELETE FROM chat_subscription cs
		USING subscription s
		WHERE s.id = cs.subscription_id
			AND cs.chat_id = $1
			AND s.url = $2
		RETURNING s.id
	`

	var linkID int64
	err = tx.QueryRow(ctx, query, chatID, url).Scan(&linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, uerrors.ErrLinkNotFound
		}

		return nil, fmt.Errorf("insert chat: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &domain.LinkWithID{ID: linkID}, nil
}

func (s *SqlLinkService) GetTimeAndUpdateLink(url string, updatedAt time.Time) (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return time.Now(), fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentUpdatedAt time.Time
	err = tx.QueryRow(ctx, `
        SELECT updated_at
        FROM subscription
        WHERE url = $1
        FOR UPDATE
    `, url).Scan(&currentUpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return time.Now(), nil
		}
		return time.Now(), fmt.Errorf("select updated_at: %w", err)
	}

	if !currentUpdatedAt.Before(updatedAt) {
		return currentUpdatedAt, nil
	}

	_, err = tx.Exec(ctx, `
        UPDATE subscription
        SET updated_at = $1
        WHERE url = $2
    `, updatedAt, url)
	if err != nil {
		return currentUpdatedAt, fmt.Errorf("update updated_at: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return currentUpdatedAt, fmt.Errorf("commit tx: %w", err)
	}

	return currentUpdatedAt, nil
}

func (s *SqlLinkService) GetLinkBatch(lastID int64) ([]domain.LinkUpdate, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
        SELECT
			subscription.id,
            subscription.url,
            subscription.updated_at,
			ARRAY_AGG(chat_subscription.chat_id) AS subscriber_ids
        FROM subscription
		INNER JOIN chat_subscription ON subscription.id = chat_subscription.subscription_id
		WHERE subscription.id > $1
        GROUP BY subscription.id, subscription.url, subscription.updated_at
		ORDER BY subscription.id
		LIMIT $2
    `

	rows, err := s.pool.Query(ctx, query, lastID, s.subscriptionBatchSize)
	if err != nil {
		return nil, 0, fmt.Errorf("query get all links: %w", err)
	}
	defer rows.Close()

	result := make([]domain.LinkUpdate, 0, rows.CommandTag().RowsAffected())
	var newLastID int64
	for rows.Next() {
		var url string
		var updatedAt time.Time
		var subscriberIDs []int64

		if err := rows.Scan(&newLastID, &url, &updatedAt, &subscriberIDs); err != nil {
			return nil, 0, fmt.Errorf("scan row: %w", err)
		}

		result = append(result, domain.LinkUpdate{
			IDs:       subscriberIDs,
			URL:       url,
			UpdatedAt: updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate rows: %w", err)
	}

	return result, newLastID, nil
}
