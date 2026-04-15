package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

type BotRestServer struct {
	bot bot.API
}

func NewBotUpdatesApi(b bot.API) *BotRestServer {
	return &BotRestServer{
		bot: b,
	}
}

func (s *BotRestServer) GetUpdate(w http.ResponseWriter, r *http.Request) {
	var req api.UpdateResponse
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSONError(w, &ErrBadRequest{err: err})
		return
	}

	w.WriteHeader(http.StatusOK)

	for _, chatID := range req.TgChatIds {
		msg := fmt.Sprintf("Получено обновление!\nСсылка: %s\n", req.Url)
		s.bot.Send(chatID, msg)
	}
}
