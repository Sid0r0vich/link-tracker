package api_test

import (
	"context"
	"fmt"
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
	"go.uber.org/mock/gomock"
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

	if err = db.Migrate(migrateCfg); err != nil {
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

type ApiTestSuite struct {
	suite.Suite
	tc  *apiTestContainers
	cfg *config.Config
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
}

func (s *ApiTestSuite) TearDownSuite() {
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

func (s *ApiTestSuite) TestApiAddLink() {
	ctx, cancel := context.WithCancel(context.Background())
	logger := logs.NewLogger()

	// scrapper init
	sqlRepo, closeDB, err := linkRepository.NewSQLRepo(&s.cfg.Database, logger)
	require.NoError(s.T(), err)
	defer func() {
		require.NoError(s.T(), closeDB())
	}()

	updateBrokerService, err := update.NewUpdateBrokerService(ctx, &s.cfg.Kafka, logger)
	require.NoError(s.T(), err)

	stackoverflowPath := "/questions/1111111111111111111111"
	stackoverflowUrl := stackoverflowPath + "/test-question"
	stackoverflowMockApi := scrapperMocks.NewMockStackoverflowAPI(s.T(), stackoverflowPath, time.Now().Unix(), time.Now().Unix(), "test body")
	defer stackoverflowMockApi.Close()
	stackOverflowScrapper := scrapper.NewStackoverflowScrapper(s.cfg.Scrapper.StackoverflowKey)
	stackOverflowScrapper.ApiHost = stackoverflowMockApi.Listener.Addr().String()
	stackOverflowScrapper.ApiScheme = "http"
	scrapperService := scrapper.NewScrapperService(map[string]scrapper.Scrapper{
		"stackoverflow.com": stackOverflowScrapper,
	})

	sched, err := scheduler.NewScheduler(sqlRepo, logger, updateBrokerService, scrapperService, time.Second)
	require.NoError(s.T(), err)
	require.NoError(s.T(), sched.Start())
	defer sched.Shutdown()

	linkService := link.NewLinkService(sqlRepo, scrapperService)
	linkService.CheckUrl = func(url string) error { return nil }
	scrapperServer := restHandlers.NewScrapperRestServer(linkService, logger, cache.NewNoCache())
	scrapperHandler := restScrapper.HandlerWithOptions(scrapperServer, restScrapper.StdHTTPServerOptions{})
	scrapperTestServer := httptest.NewServer(middleware.LoggingMiddleware(scrapperHandler, logger))
	defer scrapperTestServer.Close()

	// bot init
	stateRepo := stateRepository.NewInMemoryStateRepo()
	scrapperAdapter, err := scrapperAdapter.NewScrapperAdapterImpl(scrapperTestServer.URL)
	require.NoError(s.T(), err)
	ctrl := gomock.NewController(s.T())
	mockBotApi := chatMocks.NewMockBotApi(ctrl)
	chatController, err := chat.NewChatController(mockBotApi, scrapperAdapter, stateRepo, logger)
	require.NoError(s.T(), err)
	deliveryService := delivery.NewDeliveryService(chatController)
	require.NoError(s.T(), err)

	handler := brokerhandler.NewBotMessageHandler(deliveryService, logger)
	var wg sync.WaitGroup
	wg.Go(func() {
		broker.StartConsumerGroup(ctx, broker.NewConfig(), logger, &s.cfg.Kafka, handler.Handle)
	})

	botServer := restHandlers.NewBotRestServer(deliveryService)
	botHandler := restBot.HandlerWithOptions(botServer, restBot.StdHTTPServerOptions{})
	botTestServer := httptest.NewServer(middleware.LoggingMiddleware(botHandler, logger))
	defer botTestServer.Close()

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
			Question: "https://stackoverflow.com" + stackoverflowUrl,
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

	wg.Go(func() {
		chatController.HandleUpdates(ctx)
	})

	chatWithBot(updates, chatID, dialog)

	select {
	case <-done:
	case <-time.After(20 * time.Second):
		s.T().Fatal("timeout waiting for update notification")
	}

	cancel()
	wg.Wait()
}
