package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type BotUpdatesApi struct {
	bot application.API
}

func NewBotUpdatesApi(b application.API) *BotUpdatesApi {
	return &BotUpdatesApi{
		bot: b,
	}
}

func (api *BotUpdatesApi) GetUpdate(w http.ResponseWriter, r *http.Request) {
	var req domain.UpdateResponse
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSONError(w, &ErrBadRequest{err: err})
		return
	}

	w.WriteHeader(http.StatusOK)

	for _, chatID := range req.TgChatIds {
		msg := fmt.Sprintf("Получено обновление: %s\n%s", req.URL, req.Desc)
		api.bot.Send(chatID, msg)
	}
}
