package broker

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type stubUpdater struct {
	calls int
	last  *domain.UpdateMessage
}

func (s *stubUpdater) SendUpdate(data *domain.UpdateMessage) error {
	s.calls++
	s.last = data
	return nil
}

func newTestHandler(filtering config.FilteringConfig, updater *stubUpdater) *AgentMessageHandler {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	return NewAgentMessageHandler(updater, filtering, logger)
}

func newTestMessage(data domain.UpdateMessage) *sarama.ConsumerMessage {
	payload, _ := json.Marshal(data)
	return &sarama.ConsumerMessage{Value: payload}
}

func TestAgentMessageHandlerFiltersByStopWords(t *testing.T) {
	t.Parallel()

	updater := &stubUpdater{}
	handler := newTestHandler(config.FilteringConfig{
		StopWords: []string{"spam"},
		MinLength: 20,
	}, updater)

	handler.Handle(newTestMessage(domain.UpdateMessage{
		Id: 1,
		Data: []domain.Event{
			{Title: "useful", Description: "contains spam term", Username: "user"},
		},
	}))

	require.Zero(t, updater.calls)
	require.Nil(t, updater.last)
}

func TestAgentMessageHandlerFiltersByAuthor(t *testing.T) {
	t.Parallel()

	updater := &stubUpdater{}
	handler := newTestHandler(config.FilteringConfig{
		ExcludedAuthors: []string{"annoying-bot"},
		MinLength:       20,
	}, updater)

	handler.Handle(newTestMessage(domain.UpdateMessage{
		Id: 2,
		Data: []domain.Event{{
			Title:       "useful",
			Description: "long enough description",
			Username:    "annoying-bot",
			CreatedAt:   time.Unix(0, 0).UTC(),
		}},
	}))

	require.Zero(t, updater.calls)
	require.Nil(t, updater.last)
}

func TestAgentMessageHandlerFiltersByMinimumLength(t *testing.T) {
	t.Parallel()

	updater := &stubUpdater{}
	handler := newTestHandler(config.FilteringConfig{
		MinLength: 20,
	}, updater)

	handler.Handle(newTestMessage(domain.UpdateMessage{
		Id: 3,
		Data: []domain.Event{{
			Title:       "short",
			Description: "tiny",
			Username:    "user",
		}},
	}))

	require.Zero(t, updater.calls)
	require.Nil(t, updater.last)
}

func TestAgentMessageHandlerPassesUpdateThroughFilter(t *testing.T) {
	t.Parallel()

	updater := &stubUpdater{}
	handler := newTestHandler(config.FilteringConfig{
		StopWords:       []string{"spam"},
		ExcludedAuthors: []string{"annoying-bot"},
		MinLength:       20,
	}, updater)

	handler.Handle(newTestMessage(domain.UpdateMessage{
		Id: 4,
		Data: []domain.Event{{
			Title:       "useful title",
			Description: "useful description that is long enough",
			Username:    "user",
			CreatedAt:   time.Unix(0, 0).UTC(),
		}},
	}))

	require.Equal(t, 1, updater.calls)
	require.NotNil(t, updater.last)
	require.Len(t, updater.last.Data, 1)
	require.Equal(t, "useful title", updater.last.Data[0].Title)
	require.Equal(t, "user", updater.last.Data[0].Username)
}
