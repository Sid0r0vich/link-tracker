package update

import (
	"encoding/json"
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/broker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type UpdateBrokerService struct {
	producer *broker.Producer
	topic    string
}

func NewUpdateBrokerService(producer *broker.Producer, topic string) *UpdateBrokerService {
	return &UpdateBrokerService{producer: producer, topic: topic}
}

func (s *UpdateBrokerService) SendUpdate(data *domain.UpdateMessage) error {
	messageBytes, err := json.Marshal(*data)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	s.producer.SendMessage(s.topic, messageBytes)
	return nil
}
