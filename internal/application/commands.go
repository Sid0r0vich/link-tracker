package application

import (
	"fmt"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CmdFuncType = func(API, *tgbotapi.Message)
type CmdType struct {
	Fun  CmdFuncType
	Desc string
}

func getTextFunc(text string) CmdFuncType {
	return func(bot API, msg *tgbotapi.Message) {
		bot.Send(msg.Chat.ID, text)
	}
}

var CmdToType = map[string]CmdType{
	"start": {Fun: getTextFunc("Добро пожаловать! Используйте /help, чтобы посмотреть доступные команды."), Desc: "Начать общение"},
}
var UnknownFunc = getTextFunc("Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.")

func init() {
	CmdToType["help"] = CmdType{
		Fun: func(bot API, msg *tgbotapi.Message) {
			var keys []string
			for key := range CmdToType {
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
	Send(chatID int64, msg string)
}

func HandleCommand(bot API, msg *tgbotapi.Message) {
	cmd := strings.Split(msg.Command(), "_")

	var fun CmdFuncType
	res, ok := CmdToType[cmd[0]]
	if !ok {
		fun = UnknownFunc
	} else {
		fun = res.Fun
	}
	fun(bot, msg)
}
