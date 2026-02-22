package main

import (
	"fmt"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type cmdType = func(*tgbotapi.BotAPI, *tgbotapi.Message)

func getTextFunc(text string) cmdType {
	return func(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
	}
}

var cmdToFunc = map[string]cmdType{
	"start": getTextFunc("Добро пожаловать! Используйте /help, чтобы посмотреть доступные команды."),
}
var unknownFunc = getTextFunc("Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.")

func init() {
	cmdToFunc["help"] = func(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
		var keys []string
		for key := range cmdToFunc {
			keys = append(keys, "/"+key)
		}

		text := fmt.Sprintf(
			"Список доступных команд: %s",
			strings.Join(keys, ", "),
		)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
	}
}

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
			cmd := strings.Split(update.Message.Command(), "_")

			procFunc, ok := cmdToFunc[cmd[0]]
			if !ok {
				procFunc = unknownFunc
			}
			procFunc(bot, update.Message)
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
