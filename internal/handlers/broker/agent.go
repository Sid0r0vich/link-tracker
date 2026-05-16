package broker

import (
	"encoding/json"
	"log/slog"

	"github.com/IBM/sarama"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type Updater interface {
	SendUpdate(data *domain.UpdateMessage) error
}

type AgentMessageHandler struct {
	updateService Updater
	logger        *slog.Logger
}

func NewAgentMessageHandler(updater Updater, logger *slog.Logger) *AgentMessageHandler {
	return &AgentMessageHandler{updateService: updater, logger: logger}
}

func (s *AgentMessageHandler) Handle(msg *sarama.ConsumerMessage) {
	dataBytes := msg.Value
	var req domain.UpdateMessage
	err := json.Unmarshal(dataBytes, &req)
	if err != nil {
		s.logger.Error("unmarshaling message", slog.String("error", err.Error()))
		return
	}

	if err := s.updateService.SendUpdate(&req); err != nil {
		s.logger.Error("sending update", slog.String("error", err.Error()))
	}
}
