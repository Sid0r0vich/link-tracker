package broker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"
)

type consumerGroupHandler struct {
	ready      chan bool
	handleFunc func(message *sarama.ConsumerMessage)
}

func newConsumerGroupHandler(handleFunc func(message *sarama.ConsumerMessage)) *consumerGroupHandler {
	return &consumerGroupHandler{
		ready:      make(chan bool),
		handleFunc: handleFunc,
	}
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.handleFunc(msg)
		sess.MarkMessage(msg, "")
	}
	return nil
}

func StartConsumerGroup(
	ctx context.Context,
	cfg *sarama.Config,
	brokers []string,
	topic string,
	groupID string,
	logger *slog.Logger,
	handleFunc func(message *sarama.ConsumerMessage),
) error {
	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return fmt.Errorf("create consumer group: %w", err)
	}

	handler := newConsumerGroupHandler(handleFunc)

	for {
		if err := consumerGroup.Consume(ctx, []string{topic}, handler); err != nil {
			logger.Error("error consuming messages", slog.Any("error", err))
		}

		if ctx.Err() != nil {
			break
		}
	}

	if err := consumerGroup.Close(); err != nil {
		return fmt.Errorf("close consumer group: %w", err)
	}

	return nil

}
