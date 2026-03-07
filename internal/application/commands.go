package application

import (
	"fmt"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type cmdHandlerFunc = func(API, *tgbotapi.Message)
type CmdHandler struct {
	Fun  cmdHandlerFunc
	Desc string
}

func getTextFunc(text string) cmdHandlerFunc {
	return func(bot API, msg *tgbotapi.Message) {
		bot.Send(msg.Chat.ID, text)
	}
}

var CmdToHandler = map[string]CmdHandler{
	"start": {Fun: getTextFunc("Добро пожаловать! Используйте /help, чтобы посмотреть доступные команды."), Desc: "Начать общение"},
	"track": {Fun: func(bot API, msg *tgbotapi.Message) {
		bot.StartTrack()
	}, Desc: ""},
}
var unknownFunc = getTextFunc("Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.")

type messageHandlerFunc = func(API, *tgbotapi.Message)
type MessageHandler struct {
	Fun messageHandlerFunc
}

var StateToHandler = map[domain.BotState]MessageHandler{
	domain.StartTrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		fmt.Print("START TRACK!\n")
	}},
}
var unknownStateHandlerFunc = func(bot API, msg *tgbotapi.Message) {
	bot.Send(msg.Chat.ID, "Ошибка на стороне сервера")
}

func init() {
	CmdToHandler["help"] = CmdHandler{
		Fun: func(bot API, msg *tgbotapi.Message) {
			var keys []string
			for key := range CmdToHandler {
				keys = append(keys, "/"+key)
			}

			sort.Strings(keys)

			text := fmt.Sprintf(
				"Список доступных команд: %s",
				strings.Join(keys, ", "),
			)
			bot.Send(msg.Chat.ID, text)
		},
		Desc: "Помощь в работе с ботом",
	}
}

type API interface {
	GetState() domain.BotState
	Send(chatID int64, msg string)
	StartTrack()
}

func HandleCommand(bot API, msg *tgbotapi.Message) error {
	if !msg.IsCommand() {
		return fmt.Errorf("message is not command: %s", msg.Text)
	}

	var fun cmdHandlerFunc
	res, ok := CmdToHandler[msg.Command()]
	if !ok {
		fun = unknownFunc
	} else {
		fun = res.Fun
	}
	fun(bot, msg)

	return nil
}

func HandleMessage(bot API, msg *tgbotapi.Message) error {
	var fun messageHandlerFunc
	res, ok := StateToHandler[bot.GetState()]
	if !ok {
		fun = unknownStateHandlerFunc
	} else {
		fun = res.Fun
	}
	fun(bot, msg)

	return nil
}
