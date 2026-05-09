package utils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"slices"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"go.uber.org/fx"
)

func CheckUrl(url string, cfg *config.HTTPConfig, logger *slog.Logger) error {
	req, err := http.NewRequest("Get", url, nil)
	if err != nil {
		return fmt.Errorf("making request to bot: %w", err)
	}

	client := http.Client{Timeout: cfg.Timeout}

	return Retry(cfg, logger).Do(func() error {
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if slices.Contains(cfg.RetryableHTTPCodes, resp.StatusCode) {
			return fmt.Errorf("received retryable status code: %d", resp.StatusCode)
		}

		return nil
	})
}

func CutDescription(description string, maxLen int) string {
	return description[:min(len(description), maxLen)]
}

func IsNetError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}

func GetContext(lifecycle fx.Lifecycle) context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})

	return ctx
}
