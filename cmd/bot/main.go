package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gorilla/mux"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapter/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/middleware"
	state_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/state"
	"go.uber.org/fx"
)

type Config struct {
	BotToken      string
	TrackerAddr   string
	BotServerAddr string
}

func loadConfig(logger *slog.Logger) (*Config, error) {
	logger.Info("load config")

	botToken := os.Getenv("BOT_TOKEN")
	trackerAddr := os.Getenv("SERVER_ADDR")
	botServerAddr := os.Getenv("BOT_SERVER_ADDR")
	if botToken == "" || trackerAddr == "" || botServerAddr == "" {
		logger.Error("fail to load config", "error", "empty value")
		return nil, fmt.Errorf("environment error")
	}

	return &Config{BotToken: botToken, TrackerAddr: trackerAddr, BotServerAddr: botServerAddr}, nil
}

func startServer(cfg *Config, api *handlers.BotUpdatesApi, logger *slog.Logger) {
	r := mux.NewRouter()
	r.HandleFunc("/updates", api.GetUpdate).Methods("POST")

	err := http.ListenAndServe(cfg.BotServerAddr, middleware.LoggingMiddleware(r, logger))
	if err != nil {
		logger.Error("fail to start server", "error", err)
	}
}

func run(cfg *Config, bot *infrastructure.Bot, logger *slog.Logger, api *handlers.BotUpdatesApi) error {
	botCommands := make([]tgbotapi.BotCommand, 0, len(application.CmdToHandler))
	for name, command := range application.CmdToHandler {
		botCommands = append(botCommands, tgbotapi.BotCommand{Command: name, Description: command.Desc})
	}
	logger.Info("set command", "count", len(botCommands))
	err := bot.SetCommands(botCommands)
	if err != nil {
		logger.Error("fail to set commands", "error", err)
	}

	go func() {
		startServer(cfg, api, logger)
	}()

	updates := bot.GetUpdatesChan()
	logger.Info("get updates")
	for update := range updates {
		if update.Message == nil {
			bot.LogError(fmt.Errorf("nil message"))
			continue
		}

		if update.Message.IsCommand() {
			logger.Info("get command", "command", update.Message.Command(), "chat_id", update.Message.Chat.ID)
			_ = application.HandleCommand(bot, update.Message)
		} else {
			logger.Info("get message", "message", update.Message.Text, "chat_id", update.Message.Chat.ID)
			_ = application.HandleMessage(bot, update.Message)
		}
	}

	return nil
}

func main() {
	fx.New(
		//fx.NopLogger,
		fx.Provide(
			loadConfig,
			logs.NewLogger,
			func(cfg *Config) (*scrapper.ScrapperAdapterImpl, error) {
				return scrapper.NewScrapperAdapterImpl(fmt.Sprintf("http://%s", cfg.TrackerAddr))
			},
			state_repository.NewInMemoryStateRepo,
			func(
				cfg *Config,
				scrapperAdapter *scrapper.ScrapperAdapterImpl,
				stateRepo *state_repository.InMemoryStateRepo,
				logger *slog.Logger,
			) (*infrastructure.Bot, error) {
				return infrastructure.NewBot(cfg.BotToken, scrapperAdapter, stateRepo, logger)
			},
			func(b *infrastructure.Bot) application.API {
				return b
			},
			handlers.NewBotUpdatesApi,
		),
		fx.Invoke(run),
	).Run()
}
