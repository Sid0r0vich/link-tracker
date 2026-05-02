package link_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/db"
	orm_link_repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
)

type OrmRepoTestSuite struct {
	suite.Suite
	tc      *tcpostgres.PostgresContainer
	db      *sql.DB
	ormRepo *orm_link_repo.OrmLinkService
}

func (s *OrmRepoTestSuite) SetupSuite() {
	ctx := context.Background()

	var err error
	s.tc, err = tcpostgres.Run(
		ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("link-tracker-test-2"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(time.Minute),
		),
	)
	require.NoError(s.T(), err, "failed to start postgres container")

	connStr, err := s.tc.ConnectionString(ctx, "sslmode=disable")
	require.NoError(s.T(), err, "failed to get connection string")

	s.db, err = sql.Open("pgx", connStr)
	require.NoError(s.T(), err, "failed to create DB connection")

	migrateCfg, err := pgx.ParseConfig(connStr)
	require.NoError(s.T(), err, "failed to parse connection string for migrations")

	require.NoError(s.T(), db.Migrate(migrateCfg, "../../../db/migrations"), "failed to execute migrations")

	s.ormRepo = orm_link_repo.NewORMLinkService(s.db, SubscriptionBatchSize)
}

func (s *OrmRepoTestSuite) TearDownSuite() {
	s.db.Close()
	if err := s.tc.Terminate(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to terminate container: %v\n", err)
	}
}

func TestOrmRepoTestSuite(t *testing.T) {
	suite.Run(t, new(OrmRepoTestSuite))
}

func (s *OrmRepoTestSuite) cleanupTestDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := s.db.ExecContext(ctx, "TRUNCATE TABLE subscription_tag, chat_subscription, subscription, chat"); err != nil {
		return fmt.Errorf("failed to clean up test DB: %v", err)
	}
	return nil
}

func (s *OrmRepoTestSuite) TestOrmLinkRepo_AddAndDeleteChat() {
	LinkRepo_AddAndDeleteChatTest(s.T(), s.ormRepo)
}

func (s *OrmRepoTestSuite) TestOrmLinkRepo_GetLinksChatNotExists() {
	LinkRepo_GetLinksChatNotExistsTest(s.T(), s.ormRepo)
}

func (s *OrmRepoTestSuite) TestOrmLinkRepo_AddLinkAndGetLinks() {
	LinkRepo_AddLinkAndGetLinksTest(s.T(), s.ormRepo)
}

func (s *OrmRepoTestSuite) TestOrmLinkRepo_AddLinkChatNotExists() {
	LinkRepo_AddLinkChatNotExistsTest(s.T(), s.ormRepo)
}

func (s *OrmRepoTestSuite) TestOrmLinkRepo_AddLinkAlreadyExists() {
	LinkRepo_AddLinkAlreadyExistsTest(s.T(), s.ormRepo)
}

func (s *OrmRepoTestSuite) TestOrmLinkRepo_DeleteLink() {
	LinkRepo_DeleteLinkTest(s.T(), s.ormRepo)
}

func (s *OrmRepoTestSuite) TestOrmLinkRepo_GetTimeAndUpdateLink() {
	LinkRepo_GetTimeAndUpdateLinkTest(s.T(), s.ormRepo)
}

func (s *OrmRepoTestSuite) TestOrmLinkRepo_GetTimeAndUpdateLinkNotFound() {
	LinkRepo_GetTimeAndUpdateLinkNotFoundTest(s.T(), s.ormRepo)
}

func (s *OrmRepoTestSuite) TestOrmLinkRepo_GetLinkBatch() {
	if err := s.cleanupTestDB(); err != nil {
		s.T().Fatal(err.Error())
	}

	LinkRepo_GetLinkBatchTest(s.T(), s.ormRepo)
}
