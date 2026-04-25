package update

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/broker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type UpdateBrokerService struct {
	producer *broker.Producer
	topic    string
}

func NewUpdateBrokerService(
	ctx context.Context,
	cfg *config.KafkaConfig,
	logger *slog.Logger,
) (*UpdateBrokerService, error) {
	saramaCfg := broker.NewConfig()
	broker.CreateTopicIfNotExists(cfg, saramaCfg)
	producer, err := broker.NewProducer(ctx, saramaCfg, logger, cfg.Brokers)
	if err != nil {
		return nil, fmt.Errorf("create update producer: %w", err)
	}
	return &UpdateBrokerService{producer: producer, topic: cfg.Topic}, nil
}

func (s *UpdateBrokerService) SendUpdate(data *domain.UpdateMessage) error {
	messageBytes, err := json.Marshal(*data)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	s.producer.SendMessage(s.topic, messageBytes)
	return nil
}
