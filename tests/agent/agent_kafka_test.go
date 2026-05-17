package agent_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	mbroker "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/broker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	handlers "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/broker"
)

const testTimeout = 30 * time.Second

type readyConsumerGroupHandler struct {
	ready     chan struct{}
	readyOnce sync.Once
	handle    func(message *sarama.ConsumerMessage)
}

func (h *readyConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.readyOnce.Do(func() { close(h.ready) })
	return nil
}

func (h *readyConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *readyConsumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.handle(msg)
		sess.MarkMessage(msg, "")
	}
	return nil
}

func startConsumerGroupWithReady(
	ctx context.Context,
	cfg *sarama.Config,
	brokers []string,
	topic string,
	groupID string,
	logger *slog.Logger,
	handleFunc func(message *sarama.ConsumerMessage),
) (<-chan struct{}, <-chan error) {
	ready := make(chan struct{})
	errCh := make(chan error, 1)

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		errCh <- err
		close(ready)
		return ready, errCh
	}

	h := &readyConsumerGroupHandler{ready: ready, handle: handleFunc}

	go func() {
		defer func() {
			if closeErr := consumerGroup.Close(); closeErr != nil {
				errCh <- closeErr
				return
			}
			errCh <- nil
		}()

		for {
			if err := consumerGroup.Consume(ctx, []string{topic}, h); err != nil {
				logger.Error("error consuming messages", slog.Any("error", err))
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	return ready, errCh
}

type AgentKafkaSuite struct {
	suite.Suite
	tc      *kafka.KafkaContainer
	brokers []string
}

type stubUpdater struct {
	mu    sync.Mutex
	calls int
	byID  map[int64]int
	last  *domain.UpdateMessage
}

func (s *stubUpdater) SendUpdate(data *domain.UpdateMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byID == nil {
		s.byID = make(map[int64]int)
	}
	s.calls++
	s.byID[data.Id]++
	s.last = data
	return nil
}

func (s *stubUpdater) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func (s *stubUpdater) Last() *domain.UpdateMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

func (s *stubUpdater) UniqueIDs() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.byID)
}

func (s *stubUpdater) CountByID(id int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.byID[id]
}

func (s *AgentKafkaSuite) SetupSuite() {
	ctx := context.Background()
	kafkaContainer, err := kafka.Run(ctx, "confluentinc/confluent-local:7.6.1")
	require.NoError(s.T(), err)
	s.tc = kafkaContainer

	s.brokers, err = kafkaContainer.Brokers(ctx)
	require.NoError(s.T(), err)
}

func (s *AgentKafkaSuite) TearDownSuite() {
	if s.tc == nil {
		return
	}

	if err := s.tc.Terminate(context.Background()); err != nil {
		_, _ = os.Stderr.WriteString("failed to terminate container: " + err.Error() + "\n")
	}
}

func TestAgentKafkaSuite(t *testing.T) {
	suite.Run(t, new(AgentKafkaSuite))
}

func (s *AgentKafkaSuite) newGroupID(prefix string) string {
	return prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func (s *AgentKafkaSuite) newKafkaCfg(topic string, groupID string) (*config.KafkaConfig, *sarama.Config) {
	kafkaCfg := &config.KafkaConfig{
		Raw:               config.KafkaTopicConfig{Topic: topic, GroupID: groupID},
		Brokers:           s.brokers,
		NumPartitions:     1,
		RetentionMs:       60000,
		MinInsyncReplicas: 1,
	}

	clientCfg := mbroker.NewConfig()
	clientCfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	clientCfg.Consumer.Offsets.AutoCommit.Enable = true
	clientCfg.Consumer.Offsets.AutoCommit.Interval = 50 * time.Millisecond

	return kafkaCfg, clientCfg
}

func (s *AgentKafkaSuite) TestGetEventFromRawUpdatesOK() {
	t := s.T()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	topic := "link.raw-updates"
	groupID := s.newGroupID("agent")
	kafkaCfg, clientCfg := s.newKafkaCfg(topic, groupID)
	require.NoError(t, mbroker.CreateTopicIfNotExists(kafkaCfg.Brokers, topic, kafkaCfg, clientCfg))

	updater := &stubUpdater{}
	logBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logBuf, nil))

	h, err := handlers.NewAgentMessageHandler(
		updater,
		config.FilteringConfig{MinLength: 0},
		config.SummarizationConfig{Mode: config.SummarizationModeStub, Threshold: 0},
		logger,
	)
	require.NoError(t, err)

	received := make(chan struct{}, 1)
	ready, errCh := startConsumerGroupWithReady(ctx, clientCfg, s.brokers, topic, groupID, logger, func(message *sarama.ConsumerMessage) {
		h.Handle(message)
		if updater.Calls() > 0 {
			select {
			case received <- struct{}{}:
			default:
			}
			cancel()
		}
	})

	select {
	case <-ready:
	case <-time.After(testTimeout):
		t.Fatal("timeout waiting for consumer group readiness")
	}

	producer, err := sarama.NewSyncProducer(s.brokers, clientCfg)
	require.NoError(t, err)
	defer producer.Close()

	payload, err := json.Marshal(domain.UpdateMessage{Id: 1, Data: []domain.Event{{
		Title:       "t",
		Description: "d",
		Username:    "user",
	}}})
	require.NoError(t, err)

	_, _, err = producer.SendMessage(&sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(payload)})
	require.NoError(t, err)

	select {
	case <-received:
	case <-time.After(testTimeout):
		t.Fatal("timeout waiting for agent to process message")
	}

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(testTimeout):
		t.Fatal("timeout waiting for consumer shutdown")
	}

	require.GreaterOrEqual(t, updater.Calls(), 1)
	require.Equal(t, 1, updater.UniqueIDs())
	require.GreaterOrEqual(t, updater.CountByID(1), 1)
	require.NotNil(t, updater.Last())
	require.NotContains(t, strings.ToLower(logBuf.String()), "unmarshaling message")
}

func (s *AgentKafkaSuite) TestInvalidMessageDoesNotCrashLogsError() {
	t := s.T()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	topic := "link.raw-updates"
	groupID := s.newGroupID("agent-bad")
	kafkaCfg, clientCfg := s.newKafkaCfg(topic, groupID)
	require.NoError(t, mbroker.CreateTopicIfNotExists(kafkaCfg.Brokers, topic, kafkaCfg, clientCfg))

	updater := &stubUpdater{}
	logBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logBuf, nil))

	h, err := handlers.NewAgentMessageHandler(
		updater,
		config.FilteringConfig{MinLength: 0},
		config.SummarizationConfig{Mode: config.SummarizationModeStub, Threshold: 0},
		logger,
	)
	require.NoError(t, err)

	processed := make(chan struct{}, 1)
	ready, errCh := startConsumerGroupWithReady(ctx, clientCfg, s.brokers, topic, groupID, logger, func(message *sarama.ConsumerMessage) {
		h.Handle(message)
		if updater.Calls() > 0 {
			select {
			case processed <- struct{}{}:
			default:
			}
			cancel()
		}
	})

	select {
	case <-ready:
	case <-time.After(testTimeout):
		t.Fatal("timeout waiting for consumer group readiness")
	}

	producer, err := sarama.NewSyncProducer(s.brokers, clientCfg)
	require.NoError(t, err)
	defer producer.Close()

	_, _, err = producer.SendMessage(&sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder([]byte("{not-json"))})
	require.NoError(t, err)

	validPayload, err := json.Marshal(domain.UpdateMessage{Id: 2, Data: []domain.Event{{
		Title:       "t",
		Description: "d",
		Username:    "user",
	}}})
	require.NoError(t, err)

	_, _, err = producer.SendMessage(&sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(validPayload)})
	require.NoError(t, err)

	select {
	case <-processed:
	case <-time.After(testTimeout):
		t.Fatal("timeout waiting for agent to process valid message")
	}

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(testTimeout):
		t.Fatal("timeout waiting for consumer shutdown")
	}

	require.GreaterOrEqual(t, updater.Calls(), 1)
	require.Equal(t, 1, updater.UniqueIDs())
	require.GreaterOrEqual(t, updater.CountByID(2), 1)
	require.NotNil(t, updater.Last())
	require.Contains(t, strings.ToLower(logBuf.String()), "unmarshaling message")
}
