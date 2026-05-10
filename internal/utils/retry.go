package utils

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/avast/retry-go/v5"
	"github.com/sony/gobreaker/v2"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
)

var ErrRetry = errors.New("retryable error")

func Retry(cfg *config.HTTPConfig, logger *slog.Logger) *retry.Retrier {
	opts := []retry.Option{
		retry.Attempts(cfg.RetryCount),
		retry.OnRetry(func(n uint, err error) {
			logger.Error("retrying HTTP request...", "attempt", n+1, "error", err)
		}),
		retry.RetryIf(func(err error) bool {
			return errors.Is(err, ErrRetry)
		}),
	}

	opts = append(opts, getBackoffOption(cfg))
	return retry.New(opts...)
}

func getBackoffOption(cfg *config.HTTPConfig) retry.Option {
	strategy := cfg.RetryStrategy
	if strategy == "" {
		strategy = config.RetryStrategyConstant
	}

	switch strategy {
	case config.RetryStrategyExponential:
		return retry.DelayType(func(n uint, err error, config retry.DelayContext) time.Duration {
			delay := cfg.RetryDelay
			for i := uint(0); i < n; i++ {
				delay *= 2
			}
			return delay
		})
	default:
		return retry.Delay(cfg.RetryDelay)
	}
}

type RetryClient struct {
	c       Client
	httpCfg *config.HTTPConfig
	retry   *retry.RetrierWithData[*http.Response]
}

func NewRetryClient(c Client, cfg *config.HTTPConfig, logger *slog.Logger) *RetryClient {
	opts := []retry.Option{
		retry.Attempts(cfg.RetryCount),
		retry.OnRetry(func(n uint, err error) {
			logger.Error("retrying HTTP request...", "attempt", n+1, "error", err)
		}),
		retry.RetryIf(func(err error) bool {
			return errors.Is(err, ErrRetry)
		}),
	}

	opts = append(opts, getBackoffOption(cfg))

	return &RetryClient{
		c:       c,
		httpCfg: cfg,
		retry:   retry.NewWithData[*http.Response](opts...),
	}
}

func (c *RetryClient) Do(req *http.Request) (*http.Response, error) {
	return c.retry.Do(func() (*http.Response, error) {
		resp, err := c.c.Do(req)
		if err != nil {
			if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
				return nil, uerrors.ErrOpenState
			}
			if IsNetError(err) {
				return nil, uerrors.ErrAPIUnavailable
			}
			return nil, uerrors.ErrBadURL
		}

		if slices.Contains(c.httpCfg.RetryableHTTPCodes, resp.StatusCode) {
			return nil, fmt.Errorf("status code %d: %w", resp.StatusCode, ErrRetry)
		}

		return resp, nil
	})
}
