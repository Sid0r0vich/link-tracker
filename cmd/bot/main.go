package main

import (
	"fmt"
	"log/slog"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure"
	"go.uber.org/fx"
)

type Config struct {
	BotToken string
}

func loadConfig(logger *slog.Logger) (*Config, error) {
	logger.Info("load config")

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		logger.Error("fail to load config", "error", "empty token")
		return nil, fmt.Errorf("environment error")
	}

	return &Config{BotToken: botToken}, nil
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func run(cfg *Config, bot *infrastructure.Bot, logger *slog.Logger) error {
	botCommands := make([]tgbotapi.BotCommand, 0, len(application.CmdToType))
	for name, command := range application.CmdToType {
		botCommands = append(botCommands, tgbotapi.BotCommand{Command: name, Description: command.Desc})
	}
	logger.Info("set command", "count", len(botCommands))
	err := bot.SetCommands(botCommands)
	if err != nil {
		logger.Error("fail to set commands", "error", err)
	}

	updates := bot.GetUpdatesChan()

	logger.Info("get updates")
	for update := range updates {
		if update.Message.IsCommand() {
			logger.Info("get command", "command", update.Message.Command(), "chat_id", update.Message.Chat.ID)
			application.HandleCommand(bot.API, update.Message)
		}
	}

	return nil
}

func main() {
	fx.New(
		fx.NopLogger,
		fx.Provide(
			loadConfig,
			newLogger,
			func(cfg *Config, logger *slog.Logger) (*infrastructure.Bot, error) {
				return infrastructure.NewBot(cfg.BotToken, logger)
			},
		),
		fx.Invoke(run),
	).Run()
}
