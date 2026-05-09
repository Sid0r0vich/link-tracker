package utils

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/avast/retry-go/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
)

var ErrRetry = errors.New("retryable error")

func Retry(cfg *config.HTTPConfig, logger *slog.Logger) *retry.Retrier {
	return retry.New(
		retry.Attempts(cfg.RetryCount),
		retry.Delay(cfg.RetryDelay),
		retry.OnRetry(func(n uint, err error) {
			logger.Error("retrying HTTP request...", "attempt", n+1, "error", err)
		}),
		retry.RetryIf(func(err error) bool {
			return errors.Is(err, ErrRetry)
		}),
	)
}

type RetryClient struct {
	c       Client
	httpCfg *config.HTTPConfig
	retry   *retry.RetrierWithData[*http.Response]
}

func NewRetryClient(c Client, cfg *config.HTTPConfig, logger *slog.Logger) *RetryClient {
	return &RetryClient{
		c:       c,
		httpCfg: cfg,
		retry: retry.NewWithData[*http.Response](
			retry.Attempts(cfg.RetryCount),
			retry.Delay(cfg.RetryDelay),
			retry.OnRetry(func(n uint, err error) {
				logger.Error("retrying HTTP request...", "attempt", n+1, "error", err)
			}),
			retry.RetryIf(func(err error) bool {
				return errors.Is(err, ErrRetry)
			}),
		),
	}
}

func (c *RetryClient) Do(req *http.Request) (*http.Response, error) {
	return c.retry.Do(func() (*http.Response, error) {
		resp, err := c.c.Do(req)
		if err != nil {
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
