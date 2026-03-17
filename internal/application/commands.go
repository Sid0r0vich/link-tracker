package application

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/uerrors"
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
	"start": {
		Fun: func(bot API, msg *tgbotapi.Message) {
			err := bot.AddChat(msg.Chat.ID)
			if err != nil {
				bot.LogError(err)
			}
			bot.Send(msg.Chat.ID, "Добро пожаловать! Используйте /help, чтобы посмотреть доступные команды.")
		},
		Desc: "Начать общение",
	},
	"track": {
		Fun: func(bot API, msg *tgbotapi.Message) {
			bot.StartTrack(msg.Chat.ID)
			bot.Send(msg.Chat.ID, "Введите ссылку для трекинга:")
		},
		Desc: "Добавить ссылку для отслеживания",
	},
	"untrack": {
		Fun: func(bot API, msg *tgbotapi.Message) {
			bot.StopTrack(msg.Chat.ID)
			bot.Send(msg.Chat.ID, "Введите ссылку, которую хотите удалить:")
		},
		Desc: "Прекратить отслеживание ссылки",
	},
	"list": {
		Fun: func(bot API, msg *tgbotapi.Message) {
			message := msg.Text
			parts := strings.Split(message, " ")
			tag := ""
			if len(parts) > 1 {
				tag = parts[1]
			}

			links, err := bot.GetLinks(msg.Chat.ID, tag)
			ans := "Ссылки не найдены"
			if err != nil {
				bot.LogError(err)
				ans = "Ошибка на стороне сервера"
			}

			if len(links) > 0 {
				fmtList := make([]string, len(links))
				for ind, link := range links {
					fmtList[ind] = fmt.Sprintf(
						"Ссылка: %s\nТеги: %s\nФильтры: %s\n",
						link.URL,
						strings.Join(link.Tags, ", "),
						strings.Join(link.Filters, ", "),
					)
				}

				ans = strings.Join(fmtList, "\n")
			}

			bot.Send(msg.Chat.ID, ans)
		},
		Desc: "Получить список ссылок",
	},
	"cancel": {
		Fun: func(bot API, msg *tgbotapi.Message) {
			ans := "Отмена операции"
			if err := bot.Wait(msg.Chat.ID); err != nil {
				if errors.Is(err, ErrBotAlreadyWaiting) {

				}
				ans = "Бот ожидает команды"
			}
			bot.Send(msg.Chat.ID, ans)
		},
		Desc: "Отменить текущий флоу",
	},
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

	if err := bot.Wait(msg.Chat.ID); err != nil {
		bot.LogError(err)

		if errors.Is(err, uerrors.ErrChatNotExists) {
			err := bot.AddChat(msg.Chat.ID)
			if err != nil {
				bot.LogError(err)
			}
		}
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
