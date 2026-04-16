package rest_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/mocks"
	server "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rest"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
	"go.uber.org/mock/gomock"
)

func TestBotUpdatesApi_GetUpdate_SendsFormattedMessage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAPI := mocks.NewMockAPI(ctrl)
	h := server.NewBotUpdatesApi(mockAPI)

	chatID := int64(123)
	createdAt := time.Unix(0, 0).UTC()
	event := api.Event{
		Type:        "issue",
		Title:       "title",
		Description: "description",
		Username:    "name",
		CreatedAt:   createdAt,
	}
	reqBody := api.UpdateResponse{
		Url:       "https://github.com/sid00r/link-tracker",
		TgChatIds: []int64{chatID},
		Data:      []api.Event{event},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	expected := fmt.Sprintf(
		"Получено обновление!\nСсылка: %s\nТип: %s\nНазвание: %s\nОписание: %s\nПользователь: %s\nСоздано: %s\n",
		reqBody.Url,
		event.Type,
		event.Title,
		event.Description,
		event.Username,
		event.CreatedAt,
	)
	mockAPI.EXPECT().Send(chatID, expected).Times(1)

	r := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	h.GetUpdate(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
