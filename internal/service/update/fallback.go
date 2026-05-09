package update

import (
	"fmt"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
)

type UpdateFallbackService struct {
	rest   *UpdateRestService
	broker *UpdateBrokerService
	logger *slog.Logger
}

func NewUpdateFallbackService(rest *UpdateRestService, broker *UpdateBrokerService, logger *slog.Logger) (*UpdateFallbackService, error) {
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
	if err := s.broker.SendUpdate(data); err != nil {
		s.logger.Error("broker update failed", "error", err)
		return fmt.Errorf("rest error: %v; broker error: %w", err, err)
	}

	return nil
}
