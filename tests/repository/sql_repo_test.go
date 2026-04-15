package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/db"
	sql_link_repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link/postgres/sql"
)

type SqlRepoTestSuite struct {
	suite.Suite
	tc      *tcpostgres.PostgresContainer
	pool    *pgxpool.Pool
	sqlRepo *sql_link_repo.SqlLinkService
}

func (s *SqlRepoTestSuite) SetupSuite() {
	ctx := context.Background()

	var err error
	s.tc, err = tcpostgres.Run(
		ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("link-tracker-test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(time.Minute),
		),
	)
	if err != nil {
		panic(fmt.Errorf("failed to create test container: %v", err))
	}

	connStr, err := s.tc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(fmt.Errorf("failed to get connection string: %v", err))
	}

	s.pool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		panic(fmt.Errorf("failed to create connection pool: %v", err))
	}

	migrateCfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		panic(fmt.Errorf("failed to parse connection string for migrations: %v", err))
	}

	if err = db.Migrate(migrateCfg); err != nil {
		panic(fmt.Errorf("failed to execute migrations: %v", err))
	}

	s.sqlRepo = sql_link_repo.NewSqlLinkService(s.pool)
}

func (s *SqlRepoTestSuite) TearDownSuite() {
	s.pool.Close()
	if err := s.tc.Terminate(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to terminate container: %v\n", err)
	}
}

func TestRepoTestSuite(t *testing.T) {
	suite.Run(t, new(SqlRepoTestSuite))
}

func (s *SqlRepoTestSuite) cleanupTestDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := s.pool.Exec(ctx, "TRUNCATE TABLE subscription_tag, chat_subscription, subscription, chat"); err != nil {
		return fmt.Errorf("failed to clean up test DB: %v", err)
	}
	return nil
}

func (s *SqlRepoTestSuite) TestSqlLinkRepo_AddAndDeleteChat() {
	LinkRepo_AddAndDeleteChatTest(s.T(), s.sqlRepo)
}

func (s *SqlRepoTestSuite) TestSqlLinkRepo_GetLinksChatNotExists() {
	LinkRepo_GetLinksChatNotExistsTest(s.T(), s.sqlRepo)
}

func (s *SqlRepoTestSuite) TestSqlLinkRepo_AddLinkAndGetLinks() {
	LinkRepo_AddLinkAndGetLinksTest(s.T(), s.sqlRepo)
}

func (s *SqlRepoTestSuite) TestSqlLinkRepo_AddLinkChatNotExists() {
	LinkRepo_AddLinkChatNotExistsTest(s.T(), s.sqlRepo)
}

func (s *SqlRepoTestSuite) TestSqlLinkRepo_AddLinkAlreadyExists() {
	LinkRepo_AddLinkAlreadyExistsTest(s.T(), s.sqlRepo)
}

func (s *SqlRepoTestSuite) TestSqlLinkRepo_DeleteLink() {
	LinkRepo_DeleteLinkTest(s.T(), s.sqlRepo)
}

func (s *SqlRepoTestSuite) TestSqlLinkRepo_GetTimeAndUpdateLink() {
	LinkRepo_GetTimeAndUpdateLinkTest(s.T(), s.sqlRepo)
}

func (s *SqlRepoTestSuite) TestSqlLinkRepo_GetTimeAndUpdateLinkNotFound() {
	LinkRepo_GetTimeAndUpdateLinkNotFoundTest(s.T(), s.sqlRepo)
}

func (s *SqlRepoTestSuite) TestSqlLinkRepo_GetAllLinks() {
	if err := s.cleanupTestDB(); err != nil {
		s.T().Fatal(err.Error())
	}

	LinkRepo_GetAllLinksTest(s.T(), s.sqlRepo)
}
