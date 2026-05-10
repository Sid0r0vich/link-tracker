package update

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type stubUpdateSender struct {
	calls int
	last  *domain.UpdateMessage
	err   error
}

func (s *stubUpdateSender) SendUpdate(data *domain.UpdateMessage) error {
	s.calls++
	s.last = data
	return s.err
}

func TestUpdateFallbackServiceSendUpdateUsesBrokerOnRestTimeout(t *testing.T) {
	t.Parallel()

	restErr := context.DeadlineExceeded
	rest := &stubUpdateSender{err: restErr}
	broker := &stubUpdateSender{}
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	service, err := NewUpdateFallbackService(rest, broker, logger)
	require.NoError(t, err)

	message := &domain.UpdateMessage{
		Id:        42,
		TgChatIds: []int64{1, 2},
		Url:       "https://example.com",
		Data: []domain.Event{{
			Type:        "issue",
			Title:       "title",
			Description: "description",
			Username:    "user",
			CreatedAt:   time.Unix(0, 0).UTC(),
		}},
	}

	err = service.SendUpdate(message)
	assert.NoError(t, err)
	assert.Equal(t, 1, rest.calls)
	assert.Equal(t, 1, broker.calls)
	require.NotNil(t, broker.last)
	assert.Equal(t, message, broker.last)
	assert.Contains(t, logBuffer.String(), "rest update failed, falling back to broker")
	assert.Contains(t, logBuffer.String(), restErr.Error())
}

func TestUpdateFallbackService_SendUpdate_ReturnsBothErrorsWhenBrokerFails(t *testing.T) {
	t.Parallel()

	restErr := context.DeadlineExceeded
	brokerErr := errors.New("kafka unavailable")
	rest := &stubUpdateSender{err: restErr}
	broker := &stubUpdateSender{err: brokerErr}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	service, err := NewUpdateFallbackService(rest, broker, logger)
	require.NoError(t, err)

	err = service.SendUpdate(&domain.UpdateMessage{Url: "https://example.com"})
	require.Error(t, err)
	assert.ErrorIs(t, err, restErr)
	assert.ErrorIs(t, err, brokerErr)
	assert.True(t, strings.Contains(err.Error(), "rest error:"))
	assert.True(t, strings.Contains(err.Error(), "broker error:"))
	assert.Equal(t, 1, rest.calls)
	assert.Equal(t, 1, broker.calls)
}
