package application

import (
	"errors"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/uerrors"
)

type messageHandlerFunc = func(API, *tgbotapi.Message)
type MessageHandler struct {
	Fun messageHandlerFunc
}

func getStrs(str string) []string {
	es := strings.Split(str, ",")
	for ind, e := range es {
		es[ind] = strings.TrimSpace(e)
	}

	return es
}

var StateToHandler = map[domain.BotState]MessageHandler{
	domain.Wait: {Fun: func(bot API, msg *tgbotapi.Message) {
		bot.Send(msg.Chat.ID, "Зайдите в меню, чтобы отправить команду")
	}},
	domain.LinkTrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		err := bot.SetTrackLink(msg.Chat.ID, msg.Text)
		if err != nil {
			bot.LogError(err)
		}
		bot.Send(msg.Chat.ID, "Введите теги:")
	}},
	domain.TagsTrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		tags := getStrs(msg.Text)
		err := bot.SetTrackTags(msg.Chat.ID, tags)
		ans := "Введите фильтры:"
		if err != nil {
			bot.LogError(err)
			ans = "Данные введены некорректно"
		}
		bot.Send(msg.Chat.ID, ans)
	}},
	domain.FilterTrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		filters := getStrs(msg.Text)
		err := bot.SetTrackFilters(msg.Chat.ID, filters)
		if err != nil {
			bot.Send(msg.Chat.ID, "Данные введены некорректно")
			return
		}

		ans := "Ссылка успешно добавлена!"
		err = bot.AddLink(msg.Chat.ID)
		if err != nil {
			bot.LogError(err)
			if errors.Is(err, uerrors.ErrLinkAlreadyExists) {
				ans = "Ссылка уже отслеживается"
			}
		}
		bot.Send(msg.Chat.ID, ans)
	}},
	domain.LinkUntrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		err := bot.SetUntrackLink(msg.Chat.ID, msg.Text)
		if err != nil {
			bot.LogError(err)
		}

		ans := "Ссылка больше не отслеживается"
		err = bot.DeleteLink(msg.Chat.ID)
		if err != nil {
			ans = "Ошибка на стороне сервера"
			if errors.Is(err, uerrors.ErrChatNotExistsOrLinkNotFound) {
				ans = "Ссылка не найдена"
			} else {
				bot.LogError(err)
			}
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
		err := bot.SetData(msg.Chat.ID, data)
		if err != nil {
			bot.LogError(err)
		}
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
