package main

import (
	"fmt"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type cmdFuncType = func(*tgbotapi.BotAPI, *tgbotapi.Message)
type cmdType struct {
	fun  cmdFuncType
	desc string
}

func getTextFunc(text string) cmdFuncType {
	return func(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
	}
}

var cmdToType = map[string]cmdType{
	"start": {fun: getTextFunc("Добро пожаловать! Используйте /help, чтобы посмотреть доступные команды."), desc: "Начать общение"},
}
var unknownFunc = getTextFunc("Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.")

func init() {
	cmdToType["help"] = cmdType{
		fun: func(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
			var keys []string
			for key := range cmdToType {
				keys = append(keys, "/"+key)
			}

			text := fmt.Sprintf(
				"Список доступных команд: %s",
				strings.Join(keys, ", "),
			)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
		},
		desc: "Помощь в работе с ботом",
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

	botCommands := make([]tgbotapi.BotCommand, 0, len(cmdToType))
	for name, command := range cmdToType {
		botCommands = append(botCommands, tgbotapi.BotCommand{Command: name, Description: command.desc})
	}
	setCommandsConfig := tgbotapi.SetMyCommandsConfig{Commands: botCommands}
	if _, err := bot.Request(setCommandsConfig); err != nil {
		return err
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message.IsCommand() {
			cmd := strings.Split(update.Message.Command(), "_")

			var fun cmdFuncType
			res, ok := cmdToType[cmd[0]]
			if !ok {
				fun = unknownFunc
			} else {
				fun = res.fun
			}
			fun(bot, update.Message)
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
