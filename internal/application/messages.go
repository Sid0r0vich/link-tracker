package application

import (
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/uerrors"
)

type messageHandlerFunc = func(API, *tgbotapi.Message)
type MessageHandler struct {
	Fun messageHandlerFunc
}

var StateToHandler = map[domain.BotState]MessageHandler{
	domain.Wait: {Fun: func(bot API, msg *tgbotapi.Message) {
		bot.Send(msg.Chat.ID, "Зайдите в меню, чтобы отправить команду")
	}},
	domain.LinkTrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		bot.SetTrackLink(msg.Chat.ID, msg.Text)
		bot.Send(msg.Chat.ID, "Введите теги:")
	}},
	domain.TagsTrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		var tags []string
		err := bot.SetTrackTags(msg.Chat.ID, tags)
		ans := "Введите фильтры:"
		if err != nil {
			ans = "Данные введены некорректно"
		}
		bot.Send(msg.Chat.ID, ans)
	}},
	domain.FilterTrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		var filters []string
		err := bot.SetTrackFilters(msg.Chat.ID, filters)
		if err != nil {
			bot.Send(msg.Chat.ID, "Данные введены некорректно")
			return
		}

		ans := "Ссылка успешно добавлена!"
		err = bot.AddLink(msg.Chat.ID)
		if err != nil {
			if errors.Is(err, uerrors.ErrLinkAlreadyExists) {
				ans = "Ссылка уже отслеживается"
			}
		}
		bot.Send(msg.Chat.ID, ans)
	}},
	domain.LinkUntrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		bot.SetUntrackLink(msg.Chat.ID, msg.Text)

		ans := "Ссылка больше не отслеживается"
		err := bot.DeleteLink(msg.Chat.ID)
		if err != nil {
			bot.LogError(err)
			if errors.Is(err, uerrors.ErrLinkNotFound) {
				ans = "Ссылка не найдена"
			}

			ans = "Ошибка на стороне сервера"
		}
		bot.Send(msg.Chat.ID, ans)
	}},
}
var unknownStateHandlerFunc = func(bot API, msg *tgbotapi.Message) {
	bot.Send(msg.Chat.ID, "Ошибка на стороне сервера")
}

func HandleMessage(bot API, msg *tgbotapi.Message) error {
	var fun messageHandlerFunc
	data, err := bot.GetData(msg.Chat.ID)
	if err != nil {
		if !errors.Is(err, uerrors.ErrChatNotExists) {
			return err
		}

		data = domain.BotSimpleData{State: domain.Wait}
		bot.SetData(msg.Chat.ID, data)
	}

	res, ok := StateToHandler[data.GetState()]
	if !ok {
		fun = unknownStateHandlerFunc
	} else {
		fun = res.Fun
	}
	fun(bot, msg)

	return nil
}
