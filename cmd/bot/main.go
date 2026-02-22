package main

import (
	"fmt"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	Port = 8081
)

func startBot() error {
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		return fmt.Errorf("Environment error")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return fmt.Errorf("NewBotAPI failed: %w", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message.IsCommand() {
			var message string
			chatId := update.Message.Chat.ID
			cmd := strings.Split(update.Message.Command(), "_")

			switch cmd[0] {
			case "start":
				message = "Добро пожаловать! Используйте /help, чтобы посмотреть доступные команды."
			case "help":
				message = "Доступные команды: /start, /help"
			default:
				message = "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд."
			}

			bot.Send(tgbotapi.NewMessage(
				chatId,
				message,
			))
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
