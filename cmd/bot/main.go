package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapter/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/broker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/cache"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/chat"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	brokerhandler "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/broker"
	handlers "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rest"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/middleware"
	state_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/state"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/delivery"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
	restBot "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
	"go.uber.org/fx"
)

func startServer(cfg *config.Config, deliveryService *delivery.DeliveryService, logger *slog.Logger) {
	api := handlers.NewBotRestServer(deliveryService)

	handler := restBot.HandlerWithOptions(api, restBot.StdHTTPServerOptions{})

	err := http.ListenAndServe(cfg.Bot.ServerAddr, middleware.LoggingMiddleware(handler, logger))
	if err != nil {
		logger.Error("fail to start server", "error", err)
	}
}

func startConsumer(ctx context.Context, cfg *config.Config, deliveryService *delivery.DeliveryService, logger *slog.Logger) error {
	handler := brokerhandler.NewBotMessageHandler(deliveryService, logger)
	return broker.StartConsumerGroup(ctx, broker.NewConfig(), logger, &cfg.Kafka, handler.Handle)
}

func run(ctx context.Context, cfg *config.Config, chatController *chat.ChatController, deliveryService *delivery.DeliveryService, logger *slog.Logger) error {
	botCommands := make([]tgbotapi.BotCommand, 0, len(bot.CmdToHandler))
	for name, command := range bot.CmdToHandler {
		botCommands = append(botCommands, tgbotapi.BotCommand{Command: name, Description: command.Desc})
	}
	logger.Info("set command", "count", len(botCommands))
	err := chatController.SetCommands(botCommands)
	if err != nil {
		logger.Error("fail to set commands", "error", err)
	}

	go func() {
		switch cfg.Scrapper.UpdateCommunicationType {
		case config.UpdateCommunicationTypeHTTP:
			startServer(cfg, deliveryService, logger)

		case config.UpdateCommunicationTypeKafka:
			startConsumer(ctx, cfg, deliveryService, logger)
		}
	}()

	logger.Info("handle updates")
	chatController.HandleUpdates(ctx)

	return nil
}

func NewApp() *fx.App {
	return fx.New(
		fx.Provide(
			utils.GetContext,
			config.LoadConfig,
			logs.NewLogger,
			func(ctx context.Context, cfg *config.Config, lifecycle fx.Lifecycle, logger *slog.Logger) (scrapper.ScrapperAdapter, error) {
				var adapter scrapper.ScrapperAdapter
				switch cfg.Scrapper.TransportProtocol {
				case config.TransportProtocolHTTP:
					var err error
					adapter, err = scrapper.NewScrapperAdapterRest(fmt.Sprintf("http://%s", cfg.Scrapper.ServerAddr))
					if err != nil {
						return nil, fmt.Errorf("create REST adapter: %v", err)
					}

				case config.TransportProtocolGRPC:
					grpcAdapter, err := scrapper.NewScrapperAdapterRPC("link-tracker-scrapper:1234")
					if err != nil {
						return nil, fmt.Errorf("failed to create gRPC adapter: %v", err)
					}

					lifecycle.Append(fx.Hook{
						OnStop: func(context.Context) error {
							return grpcAdapter.ConnClose()
						},
					})

					adapter = grpcAdapter

				default:
					return nil, fmt.Errorf("invalid transport protocol: %s", cfg.Scrapper.TransportProtocol)
				}

				var clientCache cache.Cache = cache.NewNoCache()
				if cfg.Bot.CacheEnabled {
					rdb := cache.NewRedisClient(&cfg.ValKey)
					clientCache = cache.NewValKeyCache(rdb, &cfg.ValKey, "bot")
					pubsub := rdb.Subscribe(ctx, "invalidate")

					go func() {
						for msg := range pubsub.Channel() {
							key := msg.Payload
							chatID, err := strconv.ParseInt(key, 10, 64)
							if err != nil {
								logger.Error("failed to parse chatID from cache invalidation message", "key", key, "error", err)
								continue
							}

							logger.Info("invalidate cache", "key", key)
							if err := clientCache.Delete(chatID); err != nil {
								logger.Error("failed to invalidate cache", "key", key, "error", err)
							}
						}
					}()
				}

				return scrapper.NewCachedScrapperAdapter(adapter, clientCache, logger), nil
			},
			state_repository.NewInMemoryStateRepo,
			func(
				cfg *config.Config,
				scrapperAdapter scrapper.ScrapperAdapter,
				stateRepo *state_repository.InMemoryStateRepo,
				logger *slog.Logger,
			) (*chat.ChatController, error) {
				api, err := tgbotapi.NewBotAPI(cfg.Bot.Token)
				if err != nil {
					logger.Error("failed to create bot", "error", err)
					return nil, err
				}
				return chat.NewChatController(api, scrapperAdapter, stateRepo, logger)
			},
			func(b *chat.ChatController) bot.API {
				return b
			},
			delivery.NewDeliveryService,
		),
		fx.Invoke(run),
	)
}

func main() {
	NewApp().Run()
}
