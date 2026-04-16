package orm_link_repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
)

type OrmLinkService struct {
	db                    *goqu.Database
	subscriptionBatchSize uint
}

func NewORMLinkService(db *sql.DB, subscriptionBatchSize uint) *OrmLinkService {
	return &OrmLinkService{
		db:                    goqu.New("postgres", db),
		subscriptionBatchSize: subscriptionBatchSize,
	}
}

func (s *OrmLinkService) AddChat(chatID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := s.db.Insert("chat").Rows(goqu.Record{"id": chatID}).Executor().ExecContext(ctx); err != nil {
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

func (s *OrmLinkService) DeleteChat(chatID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := s.db.Delete("chat").Where(goqu.Ex{"id": chatID}).Executor().ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}

	if cnt == 0 {
		return uerrors.ErrChatNotExists
	}

	return nil
}

func (s *OrmLinkService) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Select().From("chat").Where(goqu.Ex{"id": chatID}).Executor().Exec()
	if err != nil {
		return nil, fmt.Errorf("get chat: %w", err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("get chat: %w", err)
	}

	if cnt == 0 {
		return nil, uerrors.ErrChatNotExists
	}

	links := make([]domain.DbLink, 0)
	if err := tx.
		Select(
			goqu.I("chat_subscription.id"),
			goqu.I("subscription.url"),
			goqu.I("subscription.updated_at"),
			goqu.L("ARRAY_REMOVE(ARRAY_AGG(subscription_tag.tag), NULL) ").As("tags"),
		).
		From("chat_subscription").
		Join(goqu.T("subscription"), goqu.On(goqu.Ex{"chat_subscription.subscription_id": goqu.I("subscription.id")})).
		LeftJoin(goqu.T("subscription_tag"), goqu.On(goqu.Ex{"chat_subscription.id": goqu.I("subscription_tag.chat_subscription_id")})).
		Where(goqu.Ex{"chat_subscription.chat_id": chatID}).
		GroupBy(goqu.I("chat_subscription.id"), goqu.I("subscription.url"), goqu.I("subscription.updated_at")).
		ScanStructsContext(ctx, &links); err != nil {
		return nil, fmt.Errorf("get links: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	result := make([]domain.LinkWithID, len(links))
	for i, link := range links {
		result[i] = *domain.DbLinkToLinkWithID(&link)
	}

	return result, nil
}

func (s *OrmLinkService) AddLink(chatID int64, link domain.Link) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.
		Select(goqu.I("id")).
		From("chat").
		Where(goqu.Ex{"id": chatID}).
		Executor().
		Exec()
	if err != nil {
		return 0, fmt.Errorf("query chat: %w", err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("query chat: %w", err)
	}

	if cnt == 0 {
		return 0, uerrors.ErrChatNotExists
	}

	var subscriptionID int64
	_, err = tx.Insert("subscription").
		Rows(goqu.Record{"url": link.URL, "updated_at": link.UpdatedAt}).
		OnConflict(goqu.DoUpdate("url", goqu.Record{
			"url": goqu.L("EXCLUDED.url"),
		})).
		Returning("id").
		Executor().
		ScanValContext(ctx, &subscriptionID)
	if err != nil {
		return 0, fmt.Errorf("insert subscription: %w", err)
	}

	var chatSubscriptionID int64
	_, err = tx.Insert("chat_subscription").
		Rows(goqu.Record{
			"chat_id":         chatID,
			"subscription_id": subscriptionID,
		}).
		Returning("id").
		Executor().
		ScanValContext(ctx, &chatSubscriptionID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, uerrors.ErrLinkAlreadyExists
		}
		return 0, fmt.Errorf("insert chat subscription: %w", err)
	}

	if len(link.Tags) > 0 {
		records := make([]goqu.Record, 0, len(link.Tags))
		for _, tag := range link.Tags {
			records = append(records, goqu.Record{
				"chat_subscription_id": chatSubscriptionID,
				"tag":                  tag,
			})
		}

		_, err = tx.Insert("subscription_tag").
			Rows(records).
			OnConflict(goqu.DoNothing()).
			Executor().
			ExecContext(ctx)

		if err != nil {
			return 0, fmt.Errorf("insert tags: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return chatSubscriptionID, nil
}

func (s *OrmLinkService) DeleteLink(chatID int64, url string) (*domain.LinkWithID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.
		Select(goqu.I("id")).
		From("chat").
		Where(goqu.Ex{"id": chatID}).
		Executor().
		Exec()
	if err != nil {
		return nil, fmt.Errorf("query chat: %w", err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("query chat: %w", err)
	}

	if cnt == 0 {
		return nil, uerrors.ErrChatNotExists
	}

	var subscriptionID int64
	found, err := tx.
		Select("id").
		From("subscription").
		Where(goqu.Ex{
			"url": url,
		}).
		Executor().
		ScanValContext(ctx, &subscriptionID)

	if err != nil {
		return nil, fmt.Errorf("query subscription: %w", err)
	}

	if !found {
		return nil, uerrors.ErrLinkNotFound
	}

	var linkID int64
	found, err = tx.
		Delete("chat_subscription").
		Where(
			goqu.Ex{
				"chat_subscription.chat_id": chatID,
			},
			goqu.Ex{"subscription_id": subscriptionID},
		).
		Returning("id").
		Executor().
		ScanValContext(ctx, &linkID)
	if err != nil {
		return nil, fmt.Errorf("delete link: %w", err)
	}

	if !found {
		return nil, uerrors.ErrLinkNotFound
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &domain.LinkWithID{ID: linkID}, nil
}

func (s *OrmLinkService) GetTimeAndUpdateLink(url string, updatedAt time.Time) (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return time.Now(), fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var currentUpdatedAt time.Time

	found, err := tx.
		From("subscription").
		Select("updated_at").
		Where(goqu.C("url").Eq(url)).
		ForUpdate(goqu.Wait).
		Executor().
		ScanValContext(ctx, &currentUpdatedAt)

	if err != nil {
		return time.Now(), fmt.Errorf("select updated_at: %w", err)
	}

	if !found {
		return time.Now(), fmt.Errorf("subscription not found: %s", url)
	}

	if !currentUpdatedAt.Before(updatedAt) {
		return currentUpdatedAt, nil
	}

	_, err = tx.
		Update("subscription").
		Set(goqu.Record{
			"updated_at": updatedAt,
		}).
		Where(goqu.C("url").Eq(url)).
		Executor().
		ExecContext(ctx)

	if err != nil {
		return currentUpdatedAt, fmt.Errorf("update updated_at: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return currentUpdatedAt, fmt.Errorf("commit tx: %w", err)
	}

	return currentUpdatedAt, nil
}

func (s *OrmLinkService) GetLinkBatch(lastID int64) ([]domain.LinkUpdate, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ds := s.db.
		Select(
			goqu.I("subscription.id"),
			goqu.I("subscription.url"),
			goqu.I("subscription.updated_at"),
			goqu.L("ARRAY_AGG(chat_subscription.chat_id)").As("subscriber_ids"),
		).
		From("subscription").
		Join(
			goqu.T("chat_subscription"),
			goqu.On(goqu.I("subscription.id").Eq(goqu.I("chat_subscription.subscription_id"))),
		).
		Where(goqu.I("subscription.id").Gt(lastID)).
		GroupBy(
			goqu.I("subscription.id"),
			goqu.I("subscription.url"),
			goqu.I("subscription.updated_at"),
		).
		Order(goqu.I("subscription.id").Asc()).
		Limit(s.subscriptionBatchSize)

	rows, err := ds.Executor().QueryContext(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("query get all links: %w", err)
	}
	defer rows.Close()

	result := make([]domain.LinkUpdate, 0)

	var newLastID int64
	for rows.Next() {
		var url string
		var updatedAt time.Time
		var subscriberIDs pq.Int64Array

		if err := rows.Scan(&newLastID, &url, &updatedAt, &subscriberIDs); err != nil {
			return nil, 0, fmt.Errorf("scan row: %w", err)
		}

		result = append(result, domain.LinkUpdate{
			IDs:       subscriberIDs,
			URL:       url,
			UpdatedAt: updatedAt,
		})
	}

	return result, newLastID, nil
}
