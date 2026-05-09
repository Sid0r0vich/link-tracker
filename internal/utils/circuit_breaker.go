package utils

import (
	"log/slog"
	"net/http"

	"github.com/sony/gobreaker/v2"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
)

type CircuitBreakerClient struct {
	c  Client
	cb *gobreaker.CircuitBreaker[*http.Response]
}

func NewCircuitBreakerClient(c Client, cfg *config.CircuitBreakerConfig, logger *slog.Logger) *CircuitBreakerClient {
	settings := gobreaker.Settings{
		Name:         "HTTP GET",
		MaxRequests:  cfg.PermittedCallsInHalfOpen,
		Interval:     cfg.SlidingWindowSize,
		BucketPeriod: cfg.SlidingWindowBucketSize,
		Timeout:      cfg.WaitDurationInOpenState,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < uint32(cfg.MinimumRequiredCalls) {
				return false
			}

			failurePercent := float64(counts.TotalFailures) * 100.0 / float64(counts.Requests)
			return failurePercent >= float64(cfg.FailureRateThreshold)
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			if logger != nil {
				logger.Info("circuit breaker state change", "name", name, "from", from.String(), "to", to.String())
			}
		},
	}

	return &CircuitBreakerClient{
		c:  c,
		cb: gobreaker.NewCircuitBreaker[*http.Response](settings),
	}
}

func (c *CircuitBreakerClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.cb.Execute(func() (*http.Response, error) {
		return c.c.Do(req)
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}
