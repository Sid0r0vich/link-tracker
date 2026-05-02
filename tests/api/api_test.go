package api_test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	scrapperAdapter "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapter/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/broker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/cache"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/chat"
	chatMocks "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/chat/mocks"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/db"
	brokerhandler "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/broker"
	restHandlers "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rest"
	rpcHandlers "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rpc"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/middleware"
	linkRepository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
	stateRepository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/state"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scheduler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/delivery"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper"
	scrapperMocks "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper/mocks"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/update"
	restBot "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
	restScrapper "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rest"
	scrapperRPC "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rpc"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

var (
	stackoverflowTestPath = "/questions/1111111111111111111111"
	stackoverflowTestUrl  = stackoverflowTestPath + "/test-question"
)

type apiTestContainers struct {
	kafka    *kafka.KafkaContainer
	postgres *postgres.PostgresContainer
}

func newApiTestContainers(ctx context.Context, cfg *config.Config) (*apiTestContainers, error) {
	kafcaC, err := kafka.Run(ctx, "confluentinc/cp-kafka:latest")
	if err != nil {
		return nil, err
	}

	postgresC, err := postgres.Run(
		ctx,
		"postgres:17-alpine",
		postgres.WithDatabase(cfg.Database.Name),
		postgres.WithUsername(cfg.Database.Username),
		postgres.WithPassword(cfg.Database.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(time.Minute),
		),
	)
	if err != nil {
		return nil, err
	}

	host, err := postgresC.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres host: %w", err)
	}
	port, err := postgresC.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres port: %w", err)
	}

	cfg.Database.Host = host
	cfg.Database.Port = port.Num()

	connStr, err := postgresC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %v", err)
	}

	migrateCfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string for migrations: %v", err)
	}

	if err = db.Migrate(migrateCfg, "../../db/migrations"); err != nil {
		return nil, fmt.Errorf("failed to execute migrations: %v", err)
	}

	return &apiTestContainers{kafka: kafcaC, postgres: postgresC}, nil
}

func (c *apiTestContainers) Terminate(ctx context.Context) error {
	if err := c.kafka.Terminate(ctx); err != nil {
		return fmt.Errorf("failed to terminate kafka container: %w", err)
	}

	if err := c.postgres.Terminate(ctx); err != nil {
		return fmt.Errorf("failed to terminate postgres container: %w", err)
	}

	return nil
}

func (c *apiTestContainers) ClearDB(ctx context.Context) error {
	connStr, err := c.postgres.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get connection string: %v", err)
	}

	dbConn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %v", err)
	}
	defer dbConn.Close(ctx)

	if _, err = dbConn.Exec(ctx, "TRUNCATE TABLE subscription_tag, chat_subscription, subscription, chat"); err != nil {
		return fmt.Errorf("failed to clean up test DB: %v", err)
	}

	return nil
}

type ApiTestSuite struct {
	suite.Suite
	tc                   *apiTestContainers
	cfg                  *config.Config
	logger               *slog.Logger
	stackoverflowMockApi *httptest.Server
	scrapperService      *scrapper.ScrapperService
}

func (s *ApiTestSuite) SetupSuite() {
	ctx := context.Background()

	s.cfg = &config.Config{
		Database: config.DatabaseConfig{
			Name:                  "link-tracker-test",
			Username:              "postgres",
			Password:              "postgres",
			MaxConns:              2,
			MinConns:              1,
			MaxConnIdleTimeMins:   1,
			MaxConnLifeTimeMins:   2,
			SubscriptionBatchSize: 100,
		},
		Kafka: config.KafkaConfig{
			Topic:             "link_updates",
			GroupID:           "1",
			NumPartitions:     1,
			RetentionMs:       60000,
			MinInsyncReplicas: 1,
		},
		Scrapper: config.ScrapperConfig{
			JobDelayInterval: 1 * time.Second,
		},
	}

	var err error
	s.tc, err = newApiTestContainers(ctx, s.cfg)
	require.NoError(s.T(), err)

	s.cfg.Kafka.Brokers, err = s.tc.kafka.Brokers(ctx)
	require.NoError(s.T(), err)

	s.logger = logs.NewLogger()

	s.stackoverflowMockApi = scrapperMocks.NewMockStackoverflowAPI(s.T(), stackoverflowTestPath, time.Now().Unix(), time.Now().Unix(), "test body")
	stackOverflowScrapper := scrapper.NewStackoverflowScrapper(s.cfg.Scrapper.StackoverflowKey)
	stackOverflowScrapper.ApiHost = s.stackoverflowMockApi.Listener.Addr().String()
	stackOverflowScrapper.ApiScheme = "http"
	s.scrapperService = scrapper.NewScrapperService(map[string]scrapper.Scrapper{
		"stackoverflow.com": stackOverflowScrapper,
	})
}

func (s *ApiTestSuite) TearDownSuite() {
	defer s.stackoverflowMockApi.Close()
	require.NoError(s.T(), s.tc.Terminate(context.Background()))
}

func TestApiTestSuite(t *testing.T) {
	suite.Run(t, new(ApiTestSuite))
}

func commandMessage(text string, chatID int64) *tgbotapi.Message {
	command := strings.Split(text, " ")[0]

	return &tgbotapi.Message{
		Text: text,
		Chat: &tgbotapi.Chat{ID: chatID},
		Entities: []tgbotapi.MessageEntity{
			{
				Type:   "bot_command",
				Length: len(command),
			},
		},
	}
}

type Replica struct {
	Question string
	Answer   string
}

type messageTextPrefixMatcher struct {
	chatID int64
	prefix string
}

func (m messageTextPrefixMatcher) Matches(x any) bool {
	msg, ok := x.(tgbotapi.MessageConfig)
	if !ok {
		return false
	}
	return msg.ChatID == m.chatID && strings.HasPrefix(msg.Text, m.prefix)
}

func (m messageTextPrefixMatcher) String() string {
	return fmt.Sprintf("message to chat %d with prefix %q", m.chatID, m.prefix)
}

func chatWithBot(updates chan tgbotapi.Update, chatID int64, dialog []Replica) {
	for _, replica := range dialog {
		var msg *tgbotapi.Message
		if strings.HasPrefix(replica.Question, "/") {
			msg = commandMessage(replica.Question, chatID)
		} else {
			msg = &tgbotapi.Message{
				Text: replica.Question,
				Chat: &tgbotapi.Chat{ID: chatID},
			}
		}
		updates <- tgbotapi.Update{
			Message: msg,
		}
	}
}

func (s *ApiTestSuite) TestApiAddLinkRestScrapperSqlRepositoryRestUpdater() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Require().NoError(s.tc.ClearDB(ctx))

	sqlRepo, closeSQL, err := linkRepository.NewSQLRepo(&s.cfg.Database, s.logger)
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(closeSQL())
	}()

	linkService := link.NewLinkService(sqlRepo, s.scrapperService)
	linkService.CheckUrl = func(url string) error { return nil }
	scrapperServer := restHandlers.NewScrapperRestServer(linkService, s.logger, cache.NewNoCache())
	scrapperHandler := restScrapper.HandlerWithOptions(scrapperServer, restScrapper.StdHTTPServerOptions{})
	scrapperTestServer := httptest.NewServer(middleware.LoggingMiddleware(scrapperHandler, s.logger))
	defer scrapperTestServer.Close()

	scrapperRestAdapter, err := scrapperAdapter.NewScrapperAdapterRest(scrapperTestServer.URL)
	s.Require().NoError(err)

	stateRepo := stateRepository.NewInMemoryStateRepo()
	ctrl := gomock.NewController(s.T())
	mockBotApi := chatMocks.NewMockBotApi(ctrl)
	chatController, err := chat.NewChatController(mockBotApi, scrapperRestAdapter, stateRepo, s.logger)
	s.Require().NoError(err)

	deliveryService := delivery.NewDeliveryService(chatController)
	botServer := restHandlers.NewBotRestServer(deliveryService)
	botHandler := restBot.HandlerWithOptions(botServer, restBot.StdHTTPServerOptions{})
	botTestServer := httptest.NewServer(middleware.LoggingMiddleware(botHandler, s.logger))
	defer botTestServer.Close()

	updateRestService, err := update.NewUpdateRestService(botTestServer.URL)
	s.Require().NoError(err)

	testApiAddLink(ctx, s.T(), sqlRepo, updateRestService, s.scrapperService, stackoverflowTestUrl, chatController, mockBotApi, s.logger)
}

func (s *ApiTestSuite) TestApiAddLinkRestScrapperOrmRepositoryKafkaUpdater() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Require().NoError(s.tc.ClearDB(ctx))

	ormRepo, closeORM, err := linkRepository.NewORMRepo(&s.cfg.Database, s.logger)
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(closeORM())
	}()

	linkService := link.NewLinkService(ormRepo, s.scrapperService)
	linkService.CheckUrl = func(url string) error { return nil }
	grpcLis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	scrapperRPCServer := rpcHandlers.NewScrapperRPCServer(linkService, s.logger)
	scrapperRPC.RegisterScrapperAPIServer(grpcServer, scrapperRPCServer)

	var grpcWG sync.WaitGroup
	grpcWG.Go(func() {
		_ = grpcServer.Serve(grpcLis)
	})
	defer func() {
		grpcServer.Stop()
		_ = grpcLis.Close()
		grpcWG.Wait()
	}()

	scrapperAdapterImpl, err := scrapperAdapter.NewScrapperAdapterRPC(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return grpcLis.Dial()
		}),
	)
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(scrapperAdapterImpl.ConnClose())
	}()

	stateRepo := stateRepository.NewInMemoryStateRepo()
	ctrl := gomock.NewController(s.T())
	mockBotApi := chatMocks.NewMockBotApi(ctrl)
	chatController, err := chat.NewChatController(mockBotApi, scrapperAdapterImpl, stateRepo, s.logger)
	s.Require().NoError(err)

	deliveryService := delivery.NewDeliveryService(chatController)
	handler := brokerhandler.NewBotMessageHandler(deliveryService, s.logger)

	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	updateBrokerService, err := update.NewUpdateBrokerService(consumerCtx, &s.cfg.Kafka, s.logger)
	s.Require().NoError(err)

	var wg sync.WaitGroup
	wg.Go(func() {
		_ = broker.StartConsumerGroup(consumerCtx, broker.NewConfig(), s.logger, &s.cfg.Kafka, handler.Handle)
	})

	testApiAddLink(ctx, s.T(), ormRepo, updateBrokerService, s.scrapperService, stackoverflowTestUrl, chatController, mockBotApi, s.logger)

	cancelConsumer()
	wg.Wait()
}

func testApiAddLink(
	ctx context.Context,
	t *testing.T,
	repo linkRepository.LinkUnitedRepository,
	updater scheduler.Updater,
	scrapperService *scrapper.ScrapperService,
	stackoverflowTestUrl string,
	chatController *chat.ChatController,
	mockBotApi *chatMocks.MockBotApi,
	logger *slog.Logger,
) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	// scrapper init
	sched, err := scheduler.NewScheduler(repo, logger, updater, scrapperService, time.Second)
	require.NoError(t, err)
	require.NoError(t, sched.Start())
	defer sched.Shutdown()

	// test
	chatID := int64(123)
	updates := make(chan tgbotapi.Update, 1)
	done := make(chan struct{})
	mockBotApi.EXPECT().GetUpdatesChan(gomock.Any()).Return(updates)

	dialog := []Replica{
		{
			Question: "Привет",
			Answer:   "Зайдите в меню, чтобы отправить команду",
		},
		{
			Question: "/track",
			Answer:   "Введите ссылку для трекинга:",
		},
		{
			Question: "https://stackoverflow.com" + stackoverflowTestUrl,
			Answer:   "Введите теги:",
		},
		{
			Question: "-",
			Answer:   "Введите фильтры:",
		},
		{
			Question: "-",
			Answer:   "Ссылка успешно добавлена!",
		},
		{
			Question: "/start",
			Answer:   "Добро пожаловать! Используйте /help, чтобы посмотреть доступные команды.",
		},
		{
			Question: "/help",
			Answer:   "Список доступных команд: /cancel, /help, /list, /start, /track, /untrack",
		},
	}

	for _, replica := range dialog {
		mockBotApi.EXPECT().Send(tgbotapi.NewMessage(chatID, replica.Answer))
	}

	updateCall := mockBotApi.EXPECT().Send(messageTextPrefixMatcher{chatID: chatID, prefix: "Получено обновление!"})
	updateCall.Do(func(_ tgbotapi.Chattable) {
		select {
		case <-done:
			return
		default:
			close(done)
		}
	})

	var wg sync.WaitGroup
	wg.Go(func() {
		chatController.HandleUpdates(ctx)
	})

	chatWithBot(updates, chatID, dialog)

	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("timeout waiting for update notification")
	}

	cancel()
	wg.Wait()
}
