package update

import (
	"fmt"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
)

type updateSender interface {
	SendUpdate(data *domain.UpdateMessage) error
}

type UpdateFallbackService struct {
	rest   updateSender
	broker updateSender
	logger *slog.Logger
}

func NewUpdateFallbackService(rest updateSender, broker updateSender, logger *slog.Logger) (*UpdateFallbackService, error) {
	return &UpdateFallbackService{rest: rest, broker: broker, logger: logger}, nil
}

func (s *UpdateFallbackService) SendUpdate(data *domain.UpdateMessage) error {
	err := s.rest.SendUpdate(data)
	if err == nil {
		return nil
	}

	if !utils.IsNetError(err) {
		return err
	}

	s.logger.Warn("rest update failed, falling back to broker", "error", err)
	if brokerErr := s.broker.SendUpdate(data); brokerErr != nil {
		s.logger.Error("broker update failed", "error", brokerErr)
		return fmt.Errorf("rest error: %w; broker error: %w", err, brokerErr)
	}

	return nil
}
