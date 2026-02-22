package main

import (
	"fmt"
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
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("fail to load config: %w", err)
	}

	bot, err := infrastructure.NewBot(cfg.BotToken)
	if err != nil {
		return fmt.Errorf("NewBotAPI failed: %w", err)
	}

	botCommands := make([]tgbotapi.BotCommand, 0, len(application.CmdToType))
	for name, command := range application.CmdToType {
		botCommands = append(botCommands, tgbotapi.BotCommand{Command: name, Description: command.Desc})
	}
	bot.SetCommands(botCommands)

	updates := bot.GetUpdatesChan()

	for update := range updates {
		if update.Message.IsCommand() {
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
