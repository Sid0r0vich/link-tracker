package utils

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
)

type MockClient struct {
	responses []*http.Response
	errors    []error
	callCount int
}

func NewMockClient() *MockClient {
	return &MockClient{
		responses: []*http.Response{},
		errors:    []error{},
		callCount: 0,
	}
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	defer func() { m.callCount++ }()

	if m.callCount < len(m.errors) && m.errors[m.callCount] != nil {
		return nil, m.errors[m.callCount]
	}

	if m.callCount < len(m.responses) {
		return m.responses[m.callCount], nil
	}

	return nil, errors.New("no more responses configured")
}

func newResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(nil),
		Header:     make(http.Header),
	}
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRetryClientRetryOn5xx(t *testing.T) {
	mockClient := NewMockClient()
	mockClient.responses = []*http.Response{
		newResponse(500),
		newResponse(500),
		newResponse(200),
	}

	cfg := &config.HTTPConfig{
		RetryCount:         3,
		RetryDelay:         10 * time.Millisecond,
		RetryableHTTPCodes: []int{500, 502, 503},
	}

	logger := newTestLogger()
	retryClient := NewRetryClient(mockClient, cfg, logger)

	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	resp, err := retryClient.Do(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 3, mockClient.callCount, "expected 3 attempts")
}

func TestRetryClientNoRetryOn4xx(t *testing.T) {
	mockClient := NewMockClient()
	mockClient.responses = []*http.Response{
		newResponse(400),
		newResponse(200),
	}

	cfg := &config.HTTPConfig{
		RetryCount:         3,
		RetryDelay:         10 * time.Millisecond,
		RetryableHTTPCodes: []int{500, 502, 503},
	}

	logger := newTestLogger()
	retryClient := NewRetryClient(mockClient, cfg, logger)

	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	resp, err := retryClient.Do(req)

	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, 1, mockClient.callCount, "expected 1 attempt, no retries for 4xx")
}

func TestRetryClientRetryInterval(t *testing.T) {
	mockClient := NewMockClient()
	mockClient.responses = []*http.Response{
		newResponse(500),
		newResponse(500),
		newResponse(200),
	}

	retryDelay := 50 * time.Millisecond
	cfg := &config.HTTPConfig{
		RetryCount:         3,
		RetryDelay:         retryDelay,
		RetryableHTTPCodes: []int{500, 502, 503},
	}

	logger := newTestLogger()
	retryClient := NewRetryClient(mockClient, cfg, logger)

	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	startTime := time.Now()
	resp, err := retryClient.Do(req)
	duration := time.Since(startTime)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 3, mockClient.callCount)

	minExpectedTime := 2 * retryDelay
	assert.GreaterOrEqual(t, duration, minExpectedTime,
		"expected at least 2 retry intervals")

	maxExpectedTime := minExpectedTime + 300*time.Millisecond
	assert.Less(t, duration, maxExpectedTime,
		"execution time should not exceed minimum + overhead")
}

func TestRetryClientExhaustedRetries(t *testing.T) {
	mockClient := NewMockClient()
	mockClient.responses = []*http.Response{
		newResponse(500),
		newResponse(500),
		newResponse(500),
	}

	cfg := &config.HTTPConfig{
		RetryCount:         3,
		RetryDelay:         10 * time.Millisecond,
		RetryableHTTPCodes: []int{500},
	}

	logger := newTestLogger()
	retryClient := NewRetryClient(mockClient, cfg, logger)

	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	_, err = retryClient.Do(req)

	require.Error(t, err)
	assert.Equal(t, 3, mockClient.callCount, "expected all retry attempts to be exhausted")
}

func TestRetryClientMultipleRetryableStatusCodes(t *testing.T) {
	mockClient := NewMockClient()
	mockClient.responses = []*http.Response{
		newResponse(502),
		newResponse(503),
		newResponse(200),
	}

	cfg := &config.HTTPConfig{
		RetryCount:         3,
		RetryDelay:         10 * time.Millisecond,
		RetryableHTTPCodes: []int{500, 502, 503},
	}

	logger := newTestLogger()
	retryClient := NewRetryClient(mockClient, cfg, logger)

	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	resp, err := retryClient.Do(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 3, mockClient.callCount)
}

func TestRetryClientExponentialBackoff(t *testing.T) {
	mockClient := NewMockClient()
	mockClient.responses = []*http.Response{
		newResponse(500),
		newResponse(500),
		newResponse(200),
	}

	baseDelay := 20 * time.Millisecond
	cfg := &config.HTTPConfig{
		RetryCount:         3,
		RetryDelay:         baseDelay,
		RetryableHTTPCodes: []int{500},
		RetryStrategy:      "exponential",
	}

	logger := newTestLogger()
	retryClient := NewRetryClient(mockClient, cfg, logger)

	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	startTime := time.Now()
	resp, err := retryClient.Do(req)
	duration := time.Since(startTime)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 3, mockClient.callCount)

	minExpectedTime := 3 * baseDelay
	assert.GreaterOrEqual(t, duration, minExpectedTime,
		"expected at least 3 * baseDelay for exponential backoff")

	maxExpectedTime := minExpectedTime + 150*time.Millisecond
	assert.Less(t, duration, maxExpectedTime, "execution time should not exceed minimum + overhead")
}
