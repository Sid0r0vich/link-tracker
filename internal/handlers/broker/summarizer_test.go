package broker

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

func TestStubSummarizerLongText(t *testing.T) {
	threshold := 10
	s := NewStubSummarizer(threshold)

	text := strings.Repeat("あ", threshold+5)

	out, err := s.Summarize(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(out) == strings.TrimSpace(text) {
		t.Fatalf("expected summarized output to differ from original")
	}

	if runeLen(out) > threshold+3 {
		t.Fatalf("summarized output too long: want <= %d, got %d", threshold+3, runeLen(out))
	}
}

func TestStubSummarizerShortText(t *testing.T) {
	threshold := 10
	s := NewStubSummarizer(threshold)

	text := strings.Repeat("a", threshold-1)

	out, err := s.Summarize(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(out) != strings.TrimSpace(text) {
		t.Fatalf("expected short text to be unchanged, got %q", out)
	}
}

func newTestAISummarizerWithServer(t *testing.T, response string) (*AISummarizer, *int) {
	t.Helper()

	called := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	cfg := config.SummarizationConfig{AIApiKey: "test-key", YandexFolderID: "folder", BaseURL: srv.URL}

	s, err := NewAISummarizer(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, s)

	return s, &called
}

func TestAISummarizationLongText(t *testing.T) {
	t.Parallel()

	s, called := newTestAISummarizerWithServer(t, `{"choices":[{"message":{"content":"short summary from api"}}]}`)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	threshold := 10
	updater := &stubUpdater{}
	handler := &AgentMessageHandler{
		updateService: updater,
		filtering:     config.FilteringConfig{MinLength: 0},
		summarization: config.SummarizationConfig{Mode: config.SummarizationModeAI, Threshold: threshold},
		summarizer:    s,
		logger:        logger,
	}

	longText := strings.Repeat("x", threshold+5)
	handler.Handle(newTestMessage(domain.UpdateMessage{Id: 1, Data: []domain.Event{{Title: "", Description: longText, Username: "u"}}}))

	require.Equal(t, 1, updater.calls)
	require.Equal(t, 1, *called, "expected remote API call for long text")
	require.Equal(t, "short summary from api", updater.last.Data[0].Description)
	require.NotEqual(t, longText, updater.last.Data[0].Description, "expected original text not to be passed through")
}

func TestAIShortTextNoSummarization(t *testing.T) {
	t.Parallel()

	s, called := newTestAISummarizerWithServer(t, `{"choices":[{"message":{"content":"short summary from api"}}]}`)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	threshold := 10
	updater := &stubUpdater{}
	handler := &AgentMessageHandler{
		updateService: updater,
		filtering:     config.FilteringConfig{MinLength: 0},
		summarization: config.SummarizationConfig{Mode: config.SummarizationModeAI, Threshold: threshold},
		summarizer:    s,
		logger:        logger,
	}

	shortText := strings.Repeat("x", threshold-1)
	handler.Handle(newTestMessage(domain.UpdateMessage{Id: 2, Data: []domain.Event{{Title: "", Description: shortText, Username: "u"}}}))

	require.Equal(t, 1, updater.calls)
	require.Equal(t, 0, *called, "expected no remote API call for short text")
	require.Equal(t, shortText, updater.last.Data[0].Description)
}
