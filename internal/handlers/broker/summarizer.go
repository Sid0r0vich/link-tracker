package broker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
)

type Summarizer interface {
	Summarize(text string) (string, error)
}

func newSummarizer(cfg config.SummarizationConfig, logger *slog.Logger) (Summarizer, error) {
	switch cfg.Mode {
	case config.SummarizationModeStub:
		return NewStubSummarizer(cfg.Threshold), nil
	case config.SummarizationModeAI:
		return NewAISummarizer(cfg, logger)
	default:
		return nil, fmt.Errorf("unsupported summarization mode: %s", cfg.Mode)
	}
}

type StubSummarizer struct {
	threshold int
}

func NewStubSummarizer(threshold int) *StubSummarizer {
	return &StubSummarizer{threshold: threshold}
}

func (s *StubSummarizer) Summarize(text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if s.threshold <= 0 {
		return trimmed, nil
	}

	runes := []rune(trimmed)
	if len(runes) <= s.threshold {
		return trimmed, nil
	}

	if s.threshold <= 3 {
		return string(runes[:s.threshold]), nil
	}

	return string(runes[:s.threshold]) + "...", nil
}

type AISummarizer struct {
	client         *openai.Client
	endpoint       string
	aiApiKey       string
	yandexFolderID string
	logger         *slog.Logger
}

func NewAISummarizer(cfg config.SummarizationConfig, logger *slog.Logger) (*AISummarizer, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://ai.api.cloud.yandex.net/v1"
	}

	c := openai.NewClient(
		option.WithAPIKey(cfg.AIApiKey),
		option.WithBaseURL(baseURL),
	)

	return &AISummarizer{
		client:         &c,
		aiApiKey:       cfg.AIApiKey,
		yandexFolderID: cfg.YandexFolderID,
		logger:         logger,
	}, nil
}

func (s *AISummarizer) Summarize(text string) (string, error) {
	ctx := context.Background()

	if strings.TrimSpace(text) == "" {
		return "", nil
	}

	systemPrompt := "Summarize the following update in 2–3 sentences"
	userPrompt := systemPrompt + ":\n\n" + text

	resp, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: fmt.Sprintf("gpt://%s/aliceai-llm", s.yandexFolderID),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		Temperature: openai.Float(0.3),
		MaxTokens:   openai.Int(100),
	})

	if err != nil {
		s.logger.Error("ai summarization request failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("request: %v", err)
	}

	if resp == nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from ai summarization")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func runeLen(text string) int {
	return utf8.RuneCountInString(text)
}
