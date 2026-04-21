package broker

import (
	"context"
	"log/slog"
	"sync"

	"github.com/IBM/sarama"
)

type Producer struct {
	asyncProducer sarama.AsyncProducer
	wg            *sync.WaitGroup
}

func NewProducer(
	ctx context.Context,
	cfg *sarama.Config,
	logger *slog.Logger,
	brokers []string,
) (*Producer, error) {
	producer, err := sarama.NewAsyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}

	p := &Producer{asyncProducer: producer, wg: &sync.WaitGroup{}}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		for {
			select {
			case <-ctx.Done():
				err := p.Close()
				if err != nil {
					logger.Error("error closing producer", slog.Any("error", err))
				}
				return
			case msg := <-producer.Successes():
				logger.Info("message successfully sent", slog.Int64("offset", msg.Offset))
			case errMsg := <-producer.Errors():
				logger.Error("error sending message", slog.Any("error", errMsg.Err))
			}
		}
	}()

	return p, nil
}

func (p *Producer) SendMessage(topic string, message []byte) {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(message),
	}
	p.asyncProducer.Input() <- msg
}

func (p *Producer) Close() error {
	p.wg.Wait()

	return p.asyncProducer.Close()
}
