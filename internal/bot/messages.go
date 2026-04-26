package bot

import (
	"errors"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
)

type messageHandlerFunc = func(API, *tgbotapi.Message)
type MessageHandler struct {
	Fun messageHandlerFunc
}

func getStrs(str string) []string {
	str = strings.TrimSpace(str)
	if str == "-" {
		return []string{}
	}
	data := strings.Split(str, ",")
	for i, s := range data {
		data[i] = strings.TrimSpace(s)
	}

	return data
}

var StateToHandler = map[domain.ChatState]MessageHandler{
	domain.Wait: {Fun: func(bot API, msg *tgbotapi.Message) {
		bot.Send(msg.Chat.ID, "Зайдите в меню, чтобы отправить команду")
	}},
	domain.LinkTrack: {Fun: func(bot API, msg *tgbotapi.Message) {
		err := utils.CheckLink(msg.Text)
		if err != nil {
			bot.Send(msg.Chat.ID, "Некорректная ссылка")
			return
		}

		err = bot.SetTrackLink(msg.Chat.ID, msg.Text)
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
			switch {
			case errors.Is(err, uerrors.ErrLinkAlreadyExists):
				ans = "Ссылка уже отслеживается"
			case errors.Is(err, uerrors.ErrBadURL):
				ans = "Некорректная ссылка"
			case errors.Is(err, uerrors.ErrBadToken):
				ans = "Некорректный токен"
			case errors.Is(err, uerrors.ErrTooManyRequests):
				ans = "Слишком больше количество запросов"
			case errors.Is(err, uerrors.ErrAPIUnavailable):
				ans = "Сервис не доступен"
				bot.LogError(err)
			case errors.Is(err, uerrors.ErrAPINotAlowed):
				ans = "Сервис для данного URL не поддерживается"
			default:
				ans = "Неизвестная ошибка"
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

		err := bot.AddChat(msg.Chat.ID)
		if err != nil {
			return err
		}

		data = domain.ChatSimpleData{State: domain.Wait}
		err = bot.SetData(msg.Chat.ID, data)
		if err != nil {
			bot.LogError(err)
			return err
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
