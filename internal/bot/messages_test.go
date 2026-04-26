package bot_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/mocks"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"go.uber.org/mock/gomock"
)

func plainMessage(text string) tgbotapi.Message {
	return tgbotapi.Message{
		Text: text,
		Chat: &tgbotapi.Chat{ID: 0},
	}
}

func TestHandleMessage(t *testing.T) {
	t.Run("returns error when get data fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("hello")
		mockAPI := mocks.NewMockAPI(ctrl)
		expectedErr := errors.New("storage is down")
		mockAPI.EXPECT().GetData(int64(0)).Return(nil, expectedErr)

		err := bot.HandleMessage(mockAPI, &msg)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("chat not exists initializes wait state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("hello")
		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().GetData(int64(0)).Return(nil, uerrors.ErrChatNotExists)
		mockAPI.EXPECT().AddChat(int64(0)).Return(nil)
		mockAPI.EXPECT().SetData(int64(0), gomock.Any()).DoAndReturn(func(_ int64, data domain.ChatData) error {
			if data.GetState() != domain.Wait {
				t.Fatalf("expected wait state, got %v", data.GetState())
			}
			return nil
		})
		mockAPI.EXPECT().Send(int64(0), "Зайдите в меню, чтобы отправить команду")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("chat not exists and set data error logs", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("hello")
		mockAPI := mocks.NewMockAPI(ctrl)
		setErr := errors.New("set data failed")
		mockAPI.EXPECT().GetData(int64(0)).Return(nil, uerrors.ErrChatNotExists)
		mockAPI.EXPECT().AddChat(int64(0)).Return(nil)
		mockAPI.EXPECT().SetData(int64(0), gomock.Any()).Return(setErr)
		mockAPI.EXPECT().LogError(setErr)

		err := bot.HandleMessage(mockAPI, &msg)
		if !errors.Is(err, setErr) {
			t.Fatalf("expected %v, got %v", setErr, err)
		}
	})

	t.Run("wait state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("hello")
		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.Wait}, nil)
		mockAPI.EXPECT().Send(int64(0), "Зайдите в меню, чтобы отправить команду")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("unknown state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("hello")
		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.ChatState(99)}, nil)
		mockAPI.EXPECT().Send(int64(0), "Ошибка на стороне сервера")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("link track invalid link", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("not-a-url")
		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.LinkTrack}, nil)
		mockAPI.EXPECT().Send(int64(0), "Некорректная ссылка")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("link track success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		msg := plainMessage(ts.URL)
		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.LinkTrack}, nil)
		mockAPI.EXPECT().SetTrackLink(int64(0), ts.URL).Return(nil)
		mockAPI.EXPECT().Send(int64(0), "Введите теги:")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("link track set link error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		msg := plainMessage(ts.URL)
		mockAPI := mocks.NewMockAPI(ctrl)
		setErr := errors.New("set track link failed")
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.LinkTrack}, nil)
		mockAPI.EXPECT().SetTrackLink(int64(0), ts.URL).Return(setErr)
		mockAPI.EXPECT().LogError(setErr)
		mockAPI.EXPECT().Send(int64(0), "Введите теги:")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("tags track success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("go, backend, infra")
		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.TagsTrack}, nil)
		mockAPI.EXPECT().SetTrackTags(int64(0), []string{"go", "backend", "infra"}).Return(nil)
		mockAPI.EXPECT().Send(int64(0), "Введите фильтры:")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("tags track bad data", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("go")
		mockAPI := mocks.NewMockAPI(ctrl)
		tagErr := errors.New("bad tags")
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.TagsTrack}, nil)
		mockAPI.EXPECT().SetTrackTags(int64(0), []string{"go"}).Return(tagErr)
		mockAPI.EXPECT().LogError(tagErr)
		mockAPI.EXPECT().Send(int64(0), "Данные введены некорректно")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("filter track bad filters", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("f1, f2")
		mockAPI := mocks.NewMockAPI(ctrl)
		filterErr := errors.New("bad filters")
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.FilterTrack}, nil)
		mockAPI.EXPECT().SetTrackFilters(int64(0), []string{"f1", "f2"}).Return(filterErr)
		mockAPI.EXPECT().Send(int64(0), "Данные введены некорректно")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("filter track add link success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("f1, f2")
		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.FilterTrack}, nil)
		mockAPI.EXPECT().SetTrackFilters(int64(0), []string{"f1", "f2"}).Return(nil)
		mockAPI.EXPECT().AddLink(int64(0)).Return(nil)
		mockAPI.EXPECT().Send(int64(0), "Ссылка успешно добавлена!")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("filter track add link known errors", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
			ans  string
		}{
			{name: "already exists", err: uerrors.ErrLinkAlreadyExists, ans: "Ссылка уже отслеживается"},
			{name: "bad url", err: uerrors.ErrBadURL, ans: "Некорректная ссылка"},
			{name: "bad token", err: uerrors.ErrBadToken, ans: "Некорректный токен"},
			{name: "too many requests", err: uerrors.ErrTooManyRequests, ans: "Слишком больше количество запросов"},
			{name: "external api unavailable", err: uerrors.ErrAPIUnavailable, ans: "Сервис не доступен"},
			{name: "unknown", err: errors.New("boom"), ans: "Неизвестная ошибка"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				msg := plainMessage("f1")
				mockAPI := mocks.NewMockAPI(ctrl)
				mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.FilterTrack}, nil)
				mockAPI.EXPECT().SetTrackFilters(int64(0), []string{"f1"}).Return(nil)
				mockAPI.EXPECT().AddLink(int64(0)).Return(tc.err)
				mockAPI.EXPECT().LogError(gomock.Any())
				if errors.Is(tc.err, uerrors.ErrAPIUnavailable) {
					mockAPI.EXPECT().LogError(tc.err)
				}
				mockAPI.EXPECT().Send(int64(0), tc.ans)

				if err := bot.HandleMessage(mockAPI, &msg); err != nil {
					t.Fatalf("HandleMessage returned unexpected error: %v", err)
				}
			})
		}
	})

	t.Run("link untrack success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("https://example.com")
		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.LinkUntrack}, nil)
		mockAPI.EXPECT().SetUntrackLink(int64(0), "https://example.com").Return(nil)
		mockAPI.EXPECT().DeleteLink(int64(0)).Return(nil)
		mockAPI.EXPECT().Send(int64(0), "Ссылка больше не отслеживается")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("link untrack set link error still tries delete", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("https://example.com")
		mockAPI := mocks.NewMockAPI(ctrl)
		setErr := errors.New("set untrack failed")
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.LinkUntrack}, nil)
		mockAPI.EXPECT().SetUntrackLink(int64(0), "https://example.com").Return(setErr)
		mockAPI.EXPECT().LogError(setErr)
		mockAPI.EXPECT().DeleteLink(int64(0)).Return(nil)
		mockAPI.EXPECT().Send(int64(0), "Ссылка больше не отслеживается")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("link untrack known delete error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("https://example.com")
		mockAPI := mocks.NewMockAPI(ctrl)
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.LinkUntrack}, nil)
		mockAPI.EXPECT().SetUntrackLink(int64(0), "https://example.com").Return(nil)
		mockAPI.EXPECT().DeleteLink(int64(0)).Return(uerrors.ErrChatNotExistsOrLinkNotFound)
		mockAPI.EXPECT().Send(int64(0), "Ссылка не найдена")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})

	t.Run("link untrack unknown delete error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		msg := plainMessage("https://example.com")
		mockAPI := mocks.NewMockAPI(ctrl)
		delErr := errors.New("delete failed")
		mockAPI.EXPECT().GetData(int64(0)).Return(domain.ChatSimpleData{State: domain.LinkUntrack}, nil)
		mockAPI.EXPECT().SetUntrackLink(int64(0), "https://example.com").Return(nil)
		mockAPI.EXPECT().DeleteLink(int64(0)).Return(delErr)
		mockAPI.EXPECT().LogError(delErr)
		mockAPI.EXPECT().Send(int64(0), "Ошибка на стороне сервера")

		if err := bot.HandleMessage(mockAPI, &msg); err != nil {
			t.Fatalf("HandleMessage returned unexpected error: %v", err)
		}
	})
}
