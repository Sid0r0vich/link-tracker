package rest

import (
	"encoding/json"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/delivery"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

type BotRestServer struct {
	deliveryService *delivery.DeliveryService
}

func NewBotUpdatesApi(deliveryService *delivery.DeliveryService) *BotRestServer {
	return &BotRestServer{deliveryService: deliveryService}
}

func (s *BotRestServer) GetUpdate(w http.ResponseWriter, r *http.Request) {
	var req api.UpdateResponse
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSONError(w, &ErrBadRequest{err: err})
		return
	}

	w.WriteHeader(http.StatusOK)

	s.deliveryService.MakeNewsletter(req.TgChatIds, req.Url, domain.ApiEventSliceToEventSlice(req.Data))
}
