package application

import (
	"fmt"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
		bot.Send(msg.Chat.ID, "Введите ссылку для трекинга:")
	}, Desc: "Добавить ссылку для отслеживания"},
}
var unknownFunc = getTextFunc("Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.")

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
