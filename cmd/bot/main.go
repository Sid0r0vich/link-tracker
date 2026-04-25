package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/mux"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapter/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/broker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/chat"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	brokerhandler "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/broker"
	handlers "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rest"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/middleware"
	state_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/state"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/delivery"
	"go.uber.org/fx"
)

func startServer(cfg *config.Config, deliveryService *delivery.DeliveryService, logger *slog.Logger) {
	api := handlers.NewBotUpdatesApi(deliveryService)

	r := mux.NewRouter()
	r.HandleFunc("/updates", api.GetUpdate).Methods("POST")

	err := http.ListenAndServe(cfg.Bot.ServerAddr, middleware.LoggingMiddleware(r, logger))
	if err != nil {
		logger.Error("fail to start server", "error", err)
	}
}

func startConsumer(ctx context.Context, cfg *config.Config, deliveryService *delivery.DeliveryService, logger *slog.Logger) error {
	handler := brokerhandler.NewBotMessageHandler(deliveryService, logger)
	return broker.StartConsumerGroup(ctx, broker.NewConfig(), logger, cfg.Kafka.Brokers, cfg.Kafka.GroupID, cfg.Kafka.Topic, handler.Handle)
}

func run(cfg *config.Config, chatController *chat.ChatController, deliveryService *delivery.DeliveryService, logger *slog.Logger) error {
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
			ctx, cancel := context.WithCancel(context.Background())

			sigterm := make(chan os.Signal, 1)
			signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigterm
				cancel()
			}()

			startConsumer(ctx, cfg, deliveryService, logger)
		}
	}()

	updates := chatController.GetUpdatesChan()
	logger.Info("get updates")
	for update := range updates {
		if update.Message == nil {
			chatController.LogError(fmt.Errorf("nil message"))
			continue
		}

		if update.Message.IsCommand() {
			logger.Info("get command", "command", update.Message.Command(), "chat_id", update.Message.Chat.ID)
			_ = bot.HandleCommand(chatController, update.Message)
		} else {
			logger.Info("get message", "message", update.Message.Text, "chat_id", update.Message.Chat.ID)
			_ = bot.HandleMessage(chatController, update.Message)
		}
	}

	return nil
}

func NewApp() *fx.App {
	return fx.New(
		//fx.NopLogger,
		fx.Provide(
			config.LoadConfig,
			logs.NewLogger,
			func(cfg *config.Config, lifecycle fx.Lifecycle, logger *slog.Logger) (scrapper.ScrapperAdapter, error) {
				switch cfg.Scrapper.TransportProtocol {
				case config.TransportProtocolHTTP:
					return scrapper.NewScrapperAdapterImpl(fmt.Sprintf("http://%s", cfg.Scrapper.ServerAddr))

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

					return grpcAdapter, nil
				}

				return nil, fmt.Errorf("invalid transport protocol: %s", cfg.Scrapper.TransportProtocol)
			},
			state_repository.NewInMemoryStateRepo,
			func(
				cfg *config.Config,
				scrapperAdapter scrapper.ScrapperAdapter,
				stateRepo *state_repository.InMemoryStateRepo,
				logger *slog.Logger,
			) (*chat.ChatController, error) {
				return chat.NewChatController(cfg.Bot.Token, scrapperAdapter, stateRepo, logger)
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
