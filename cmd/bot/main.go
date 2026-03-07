package main

import (
	"fmt"
	"log/slog"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	"go.uber.org/fx"
)

type Config struct {
	BotToken    string
	TrackerAddr string
}

func loadConfig(logger *slog.Logger) (*Config, error) {
	logger.Info("load config")

	botToken := os.Getenv("BOT_TOKEN")
	trackerAddr := os.Getenv("SERVER_ADDR")
	if botToken == "" || trackerAddr == "" {
		logger.Error("fail to load config", "error", "empty value")
		return nil, fmt.Errorf("environment error")
	}

	return &Config{BotToken: botToken, TrackerAddr: trackerAddr}, nil
}

func run(cfg *Config, bot *infrastructure.Bot, logger *slog.Logger) error {
	botCommands := make([]tgbotapi.BotCommand, 0, len(application.CmdToHandler))
	for name, command := range application.CmdToHandler {
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
			func(cfg *Config) *infrastructure.Scrapper {
				return infrastructure.NewScrapper(cfg.TrackerAddr)
			},
			func(cfg *Config, tracker *infrastructure.Scrapper, logger *slog.Logger) (*infrastructure.Bot, error) {
				return infrastructure.NewBot(cfg.BotToken, tracker, logger)
			},
		),
		fx.Invoke(run),
	).Run()
}
