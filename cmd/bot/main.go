package main

import (
	"fmt"
	"log/slog"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure"
)

type Config struct {
	BotToken string
}

func loadConfig() (*Config, error) {
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		return nil, fmt.Errorf("environment error")
	}

	return &Config{BotToken: botToken}, nil
}

func startBot() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	logger.Info("load config")
	cfg, err := loadConfig()
	if err != nil {
		logger.Error("fail to load config", "error", err)
		return fmt.Errorf("fail to load config: %w", err)
	}

	logger.Info("init bot")
	bot, err := infrastructure.NewBot(cfg.BotToken)
	if err != nil {
		logger.Error("fail to create bot", "error", err)
		return fmt.Errorf("NewBotAPI failed: %w", err)
	}

	botCommands := make([]tgbotapi.BotCommand, 0, len(application.CmdToType))
	for name, command := range application.CmdToType {
		botCommands = append(botCommands, tgbotapi.BotCommand{Command: name, Description: command.Desc})
	}
	logger.Info("set command", "count", len(botCommands))
	err = bot.SetCommands(botCommands)
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
	err := startBot()
	if err != nil {
		panic(err)
	}
}
