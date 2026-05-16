package broker

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/IBM/sarama"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type Updater interface {
	SendUpdate(data *domain.UpdateMessage) error
}

type AgentMessageHandler struct {
	updateService Updater
	filtering     config.FilteringConfig
	logger        *slog.Logger
}

func NewAgentMessageHandler(updater Updater, filtering config.FilteringConfig, logger *slog.Logger) *AgentMessageHandler {
	return &AgentMessageHandler{updateService: updater, filtering: filtering, logger: logger}
}

func (s *AgentMessageHandler) Handle(msg *sarama.ConsumerMessage) {
	dataBytes := msg.Value
	var req domain.UpdateMessage
	err := json.Unmarshal(dataBytes, &req)
	if err != nil {
		s.logger.Error("unmarshaling message", slog.String("error", err.Error()))
		return
	}

	req.Data = s.filterEvents(req.Data)
	if len(req.Data) == 0 {
		s.logger.Info("update filtered out")
		return
	}

	if err := s.updateService.SendUpdate(&req); err != nil {
		s.logger.Error("sending update", slog.String("error", err.Error()))
	}
}

func (s *AgentMessageHandler) filterEvents(events []domain.Event) []domain.Event {
	filtered := make([]domain.Event, 0, len(events))
	for _, event := range events {
		if s.shouldSkipEvent(event) {
			continue
		}

		filtered = append(filtered, event)
	}

	return filtered
}

func (s *AgentMessageHandler) shouldSkipEvent(event domain.Event) bool {
	if s.isExcludedAuthor(event.Username) {
		return true
	}

	text := strings.ToLower(strings.TrimSpace(event.Title + " " + event.Description))
	if len(text) < s.filtering.MinLength {
		return true
	}

	for _, stopWord := range s.filtering.StopWords {
		if stopWord == "" {
			continue
		}

		if strings.Contains(text, strings.ToLower(stopWord)) {
			return true
		}
	}

	return false
}

func (s *AgentMessageHandler) isExcludedAuthor(username string) bool {
	for _, excludedAuthor := range s.filtering.ExcludedAuthors {
		if excludedAuthor == "" {
			continue
		}

		if strings.EqualFold(username, excludedAuthor) {
			return true
		}
	}

	return false
}
