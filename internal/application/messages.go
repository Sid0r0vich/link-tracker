package application

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type messageHandlerFunc = func(API, *tgbotapi.Message)
type MessageHandler struct {
	Fun messageHandlerFunc
}

var StateToHandler = map[domain.BotState]MessageHandler{
	domain.Wait: {Fun: func(bot API, msg *tgbotapi.Message) {
		bot.Send(msg.Chat.ID, "Зайдите в меню, чтобы отправить команду")
	}},
	domain.StartTrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		bot.SetTrackLink(msg.Text)
		bot.Send(msg.Chat.ID, "Введите теги:")
	}},
	domain.LinkTrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		var tags []string
		err := bot.SetTrackTags(tags)
		ans := "Введите фильтры:"
		if err != nil {
			ans = "Данные введены некорректно"
		}
		bot.Send(msg.Chat.ID, ans)
	}},
	domain.FilterTrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		var filters []string
		err := bot.SetTrackFilters(filters)
		if err != nil {
			bot.Send(msg.Chat.ID, "Данные введены некорректно")
			return
		}

		ans := "Ссылка успешно добавлена!"
		err = bot.AddLink(msg.Chat.ID)
		if err != nil {
			bot.Send(msg.Chat.ID, "Внутренняя ошибка")
			return
		}
		bot.Send(msg.Chat.ID, ans)
	}},
}
var unknownStateHandlerFunc = func(bot API, msg *tgbotapi.Message) {
	bot.Send(msg.Chat.ID, "Ошибка на стороне сервера")
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
