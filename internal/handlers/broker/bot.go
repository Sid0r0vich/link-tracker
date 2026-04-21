package broker

import (
	"encoding/json"
	"log/slog"

	"github.com/IBM/sarama"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/delivery"
)

type BotMessageHandler struct {
	deliveryService *delivery.DeliveryService
	logger          *slog.Logger
}

func NewBotMessageHandler(deliveryService *delivery.DeliveryService, logger *slog.Logger) *BotMessageHandler {
	return &BotMessageHandler{deliveryService: deliveryService, logger: logger}
}

func (s *BotMessageHandler) Handle(msg *sarama.ConsumerMessage) {
	dataBytes := msg.Value
	var req domain.UpdateMessage
	err := json.Unmarshal(dataBytes, &req)
	if err != nil {
		s.logger.Error("unmarshaling message", slog.String("error", err.Error()))
		return
	}

	s.deliveryService.MakeNewsletter(req.TgChatIds, req.Url, req.Data)
}
