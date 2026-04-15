package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/mux"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapter/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/chat"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	handlers "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rest"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/middleware"
	state_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/state"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func startServer(cfg *config.Config, api *handlers.BotRestServer, logger *slog.Logger) {
	r := mux.NewRouter()
	r.HandleFunc("/updates", api.GetUpdate).Methods("POST")

	err := http.ListenAndServe(cfg.Bot.ServerAddr, middleware.LoggingMiddleware(r, logger))
	if err != nil {
		logger.Error("fail to start server", "error", err)
	}
}

func run(cfg *config.Config, chatController *chat.ChatController, logger *slog.Logger, api *handlers.BotRestServer) error {
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
		startServer(cfg, api, logger)
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

func main() {
	fx.New(
		//fx.NopLogger,
		fx.Provide(
			config.LoadConfig,
			logs.NewLogger,
			func(cfg *config.Config, lifecycle fx.Lifecycle, logger *slog.Logger) (scrapper.ScrapperAdapter, error) {
				switch cfg.Scrapper.TransportProtocol {
				case config.TransportProtocolHTTP:
					return scrapper.NewScrapperAdapterImpl(fmt.Sprintf("http://%s", cfg.Scrapper.ServerAddr))

				case config.TransportProtocolGRPC:
					conn, err := grpc.NewClient("link-tracker-scrapper:1234", grpc.WithTransportCredentials(insecure.NewCredentials()))
					if err != nil {
						return nil, fmt.Errorf("failed to connect to scrapper: %v", err)
					}

					lifecycle.Append(fx.Hook{
						OnStop: func(context.Context) error {
							conn.Close()
							logger.Info("grpc connection closed")
							return nil
						},
					})

					return scrapper.NewScrapperAdapterRPC(conn)
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
			handlers.NewBotUpdatesApi,
		),
		fx.Invoke(run),
	).Run()
}
