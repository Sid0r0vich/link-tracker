package link

import (
	"database/sql"
	"fmt"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/db"
)

func NewSQLRepo(
	cfg *config.DatabaseConfig,
	logger *slog.Logger,
) (LinkUnitedRepository, func() error, error) {
	pgxCfg, err := db.GetConnCfg(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("get pgx pool config: %w", err)
	}

	pool, err := db.GetDBPoolConn(pgxCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to db: %w", err)
	}

	return NewSqlLinkService(pool, cfg.SubscriptionBatchSize), func() error {
		db.CloseDBConn()
		return nil
	}, nil
}

func NewORMRepo(
	cfg *config.DatabaseConfig,
	logger *slog.Logger,
) (LinkUnitedRepository, func() error, error) {
	db, err := sql.Open("pgx", db.GetDSNFromConfig(cfg))
	if err != nil {
		return nil, nil, fmt.Errorf("fail to open database: %v", err)
	}

	return NewORMLinkService(db, cfg.SubscriptionBatchSize), db.Close, nil
}
