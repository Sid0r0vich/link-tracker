package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
)

var once sync.Once
var pgPool *pgxpool.Pool

// Оставил эту функцию для тестов
func Migrate(cfg *pgx.ConnConfig) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	db := stdlib.OpenDB(*cfg)
	defer func() { _ = db.Close() }()

	if err := goose.Up(db, "../../db/migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}

func GetDSNFromConfig(cfg *config.DatabaseConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?target_session_attrs=read-write&sslmode=disable",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
	)
}

func GetConnCfg(cfg *config.DatabaseConfig) (*pgxpool.Config, error) {
	dsn := GetDSNFromConfig(cfg)

	connCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}

	connCfg.MaxConns = int32(cfg.MaxConns)
	connCfg.MinConns = int32(cfg.MinConns)
	connCfg.MaxConnIdleTime = time.Duration(cfg.MaxConnIdleTimeMins) * time.Minute
	connCfg.MaxConnLifetime = time.Duration(cfg.MaxConnLifeTimeMins) * time.Minute

	return connCfg, nil
}

func GetDBPoolConn(connCfg *pgxpool.Config) (*pgxpool.Pool, error) {
	var err error

	once.Do(func() {
		ctx := context.Background()

		pgPool, err = pgxpool.NewWithConfig(ctx, connCfg)
		if err != nil {
			err = fmt.Errorf("create pool: %w", err)
			return
		}

		err = pgPool.Ping(ctx)
		if err != nil {
			err = fmt.Errorf("ping DB: %w", err)
			return
		}
	})

	if err != nil {
		return nil, err
	}

	return pgPool, nil
}

func CloseDBConn() {
	pgPool.Close()
}
