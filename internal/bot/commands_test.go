package bot_test

import (
	"errors"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/mocks"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"go.uber.org/mock/gomock"
)

func commandMessage(text string) tgbotapi.Message {
	command := strings.Split(text, " ")[0]

	return tgbotapi.Message{
		Text: text,
		Chat: &tgbotapi.Chat{ID: 0},
		Entities: []tgbotapi.MessageEntity{
			{
				Type:   "bot_command",
				Length: len(command),
			},
		},
	}
}

func TestHandleCommands(t *testing.T) {
	t.Run("start command", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().Wait(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().AddChat(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().Send(int64(0), "Добро пожаловать! Используйте /help, чтобы посмотреть доступные команды.").Times(1)

		msg := commandMessage("/start")
		if err := bot.HandleCommand(mockAPI, &msg); err != nil {
			t.Fatalf("HandleCommand returned unexpected error: %v", err)
		}
	})

	t.Run("help command", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().Wait(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().Send(int64(0), "Список доступных команд: /cancel, /help, /list, /start, /track, /untrack").Times(1)

		msg := commandMessage("/help")
		if err := bot.HandleCommand(mockAPI, &msg); err != nil {
			t.Fatalf("HandleCommand returned unexpected error: %v", err)
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().Wait(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().Send(int64(0), "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.").Times(1)

		msg := commandMessage("/unknown")
		if err := bot.HandleCommand(mockAPI, &msg); err != nil {
			t.Fatalf("HandleCommand returned unexpected error: %v", err)
		}
	})

	t.Run("track command", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().Wait(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().StartTrack(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().Send(int64(0), "Введите ссылку для трекинга:").Times(1)

		msg := commandMessage("/track")
		if err := bot.HandleCommand(mockAPI, &msg); err != nil {
			t.Fatalf("HandleCommand returned unexpected error: %v", err)
		}
	})

	t.Run("untrack command", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().Wait(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().StopTrack(int64(0)).Times(1)
		mockAPI.EXPECT().Send(int64(0), "Введите ссылку, которую хотите удалить:").Times(1)

		msg := commandMessage("/untrack")
		if err := bot.HandleCommand(mockAPI, &msg); err != nil {
			t.Fatalf("HandleCommand returned unexpected error: %v", err)
		}
	})

	t.Run("list command without links", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().Wait(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().GetLinks(int64(0), "").Return(nil, nil).Times(1)
		mockAPI.EXPECT().Send(int64(0), "Ссылки не найдены").Times(1)

		msg := commandMessage("/list")
		if err := bot.HandleCommand(mockAPI, &msg); err != nil {
			t.Fatalf("HandleCommand returned unexpected error: %v", err)
		}
	})

	t.Run("list command with tag and links", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPI := mocks.NewMockAPI(ctrl)
		links := []domain.LinkWithID{
			{
				Link: domain.Link{
					URL: "https://example.com/1",
					LinkInfo: domain.LinkInfo{
						Tags:    []string{"go", "backend"},
						Filters: []string{"author=alice", "score>10"},
					},
				},
				ID: 1,
			},
			{
				Link: domain.Link{
					URL: "https://example.com/2",
					LinkInfo: domain.LinkInfo{
						Tags:    []string{"devops"},
						Filters: []string{"team=infra"},
					},
				},
				ID: 2,
			},
		}

		expected := "Ссылка: https://example.com/1\nТеги: go, backend\nФильтры: author=alice, score>10\n\nСсылка: https://example.com/2\nТеги: devops\nФильтры: team=infra\n"

		mockAPI.EXPECT().Wait(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().GetLinks(int64(0), "go").Return(links, nil).Times(1)
		mockAPI.EXPECT().Send(int64(0), expected).Times(1)

		msg := commandMessage("/list go")
		if err := bot.HandleCommand(mockAPI, &msg); err != nil {
			t.Fatalf("HandleCommand returned unexpected error: %v", err)
		}
	})

	t.Run("list command repository error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPI := mocks.NewMockAPI(ctrl)
		repoErr := errors.New("repo error")
		mockAPI.EXPECT().Wait(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().GetLinks(int64(0), "").Return(nil, repoErr).Times(1)
		mockAPI.EXPECT().LogError(repoErr).Times(1)
		mockAPI.EXPECT().Send(int64(0), "Ошибка на стороне сервера").Times(1)

		msg := commandMessage("/list")
		if err := bot.HandleCommand(mockAPI, &msg); err != nil {
			t.Fatalf("HandleCommand returned unexpected error: %v", err)
		}
	})

	t.Run("cancel command", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().Wait(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().Wait(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().Send(int64(0), "Отмена операции").Times(1)

		msg := commandMessage("/cancel")
		if err := bot.HandleCommand(mockAPI, &msg); err != nil {
			t.Fatalf("HandleCommand returned unexpected error: %v", err)
		}
	})

	t.Run("cancel command when bot already waiting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().Wait(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().Wait(int64(0)).Return(uerrors.ErrBotAlreadyWaiting).Times(1)
		mockAPI.EXPECT().Send(int64(0), "Бот ожидает команды").Times(1)

		msg := commandMessage("/cancel")
		if err := bot.HandleCommand(mockAPI, &msg); err != nil {
			t.Fatalf("HandleCommand returned unexpected error: %v", err)
		}
	})

	t.Run("auto add chat on wait chat-not-exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().Wait(int64(0)).Return(uerrors.ErrChatNotExists).Times(1)
		mockAPI.EXPECT().AddChat(int64(0)).Return(nil).Times(1)
		mockAPI.EXPECT().Send(int64(0), "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд.").Times(1)

		msg := commandMessage("/unknown")
		if err := bot.HandleCommand(mockAPI, &msg); err != nil {
			t.Fatalf("HandleCommand returned unexpected error: %v", err)
		}
	})
}
