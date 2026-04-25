package broker_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/broker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
)

type KafkaTestSuite struct {
	suite.Suite
	tc      *kafka.KafkaContainer
	brokers []string
}

func (s *KafkaTestSuite) SetupSuite() {
	ctx := context.Background()

	kafkaContainer, err := kafka.Run(ctx, "confluentinc/cp-kafka:latest")
	require.NoError(s.T(), err)
	s.tc = kafkaContainer

	s.brokers, err = kafkaContainer.Brokers(ctx)
	require.NoError(s.T(), err)
}

func (s *KafkaTestSuite) TearDownSuite() {
	if s.tc == nil {
		return
	}

	if err := s.tc.Terminate(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to terminate container: %v\n", err)
	}
}

func TestKafkaTestSuite(t *testing.T) {
	suite.Run(t, new(KafkaTestSuite))
}

func (s *KafkaTestSuite) newTopicName(prefix string) string {
	return prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func (s *KafkaTestSuite) TestCreateTopicIfNotExists_Idempotent() {
	topic := s.newTopicName("links")

	kafkaCfg := config.KafkaConfig{
		Brokers:           s.brokers,
		Topic:             topic,
		NumPartitions:     1,
		RetentionMs:       60000,
		MinInsyncReplicas: 1,
	}
	clientCfg := broker.NewConfig(nil)

	require.NoError(s.T(), broker.CreateTopicIfNotExists(kafkaCfg, clientCfg))
	require.NoError(s.T(), broker.CreateTopicIfNotExists(kafkaCfg, clientCfg))

	admin, err := sarama.NewClusterAdmin(s.brokers, clientCfg)
	require.NoError(s.T(), err)
	defer admin.Close()

	topics, err := admin.ListTopics()
	require.NoError(s.T(), err)

	detail, exists := topics[topic]
	assert.True(s.T(), exists)
	assert.EqualValues(s.T(), 1, detail.NumPartitions)
	assert.EqualValues(s.T(), int16(len(s.brokers)), detail.ReplicationFactor)
}

func (s *KafkaTestSuite) TestStartConsumerGroup_ConsumesMessage() {
	topic := s.newTopicName("updates")
	groupID := s.newTopicName("group")
	payload := []byte("integration-message")

	kafkaCfg := config.KafkaConfig{
		Brokers:           s.brokers,
		Topic:             topic,
		NumPartitions:     1,
		RetentionMs:       60000,
		MinInsyncReplicas: 1,
	}
	clientCfg := broker.NewConfig(nil, func(cfg *sarama.Config) {
		cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	})
	require.NoError(s.T(), broker.CreateTopicIfNotExists(kafkaCfg, clientCfg))

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	s.T().Cleanup(cancelConsumer)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	errCh := make(chan error, 1)
	received := make(chan *sarama.ConsumerMessage, 1)

	go func() {
		errCh <- broker.StartConsumerGroup(
			consumerCtx,
			clientCfg,
			logger,
			s.brokers,
			groupID,
			topic,
			func(message *sarama.ConsumerMessage) {
				select {
				case received <- message:
				default:
				}
				cancelConsumer()
			},
		)
	}()

	producer, err := sarama.NewSyncProducer(s.brokers, clientCfg)
	require.NoError(s.T(), err)
	defer producer.Close()

	_, _, err = producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(payload),
	})
	require.NoError(s.T(), err)

	select {
	case msg := <-received:
		require.NotNil(s.T(), msg)
		assert.Equal(s.T(), topic, msg.Topic)
		assert.Equal(s.T(), payload, msg.Value)
	case <-time.After(10 * time.Second):
		s.T().Fatal("timeout waiting for consumed message")
	}

	select {
	case err := <-errCh:
		require.NoError(s.T(), err)
	case <-time.After(10 * time.Second):
		s.T().Fatal("timeout waiting for consumer group shutdown")
	}
}
