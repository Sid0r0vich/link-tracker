package scrapper_test

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper/mocks"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

type GithubSuite struct {
	suite.Suite
	ts  *httptest.Server
	cfg *mocks.ApiConfig
	cb  *config.CircuitBreakerConfig
}

func (s *GithubSuite) SetupTest() {
	createdAt := time.Unix(0, 0).UTC()
	body := strings.Repeat("b", scrapper.MaxDescriptionLength+10)

	s.cfg = &mocks.ApiConfig{
		ServerUrl:   "/repos",
		OkPath:      "/acme/ok",
		TimeoutPath: "/acme/timeout",
		FailPath:    "/acme/fail",
		Timeout:     time.Millisecond * 200,
		Body:        body,
	}
	s.cb = &config.CircuitBreakerConfig{
		SlidingWindowSize:        500 * time.Millisecond,
		SlidingWindowBucketSize:  100 * time.Millisecond,
		MinimumRequiredCalls:     2,
		FailureRateThreshold:     51,
		PermittedCallsInHalfOpen: 4,
		WaitDurationInOpenState:  100 * time.Millisecond,
	}
	s.ts = mocks.NewMockGithubAPI(s.T(), s.cfg, createdAt)
}

func (s *GithubSuite) TearDownTest() {
	s.ts.Close()
}

func TestGithubSuite(t *testing.T) {
	suite.Run(t, new(GithubSuite))
}

func (s *GithubSuite) TestGetUpdateNewIssueFormatsEvent() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	scr := scrapper.NewGithubScrapper(&config.HTTPConfig{Timeout: 5 * time.Second}, &config.CircuitBreakerConfig{}, "test-token", logger)
	scr.ApiScheme = "http"
	scr.ApiHost = s.ts.Listener.Addr().String()
	scr.Client = s.ts.Client()

	updateUrl := "https://github.com" + s.cfg.OkPath
	upd, err := scr.GetUpdate(updateUrl)
	s.NoError(err)
	s.NotNil(upd)
	s.Equal(updateUrl, upd.URL)
	s.Equal(1, len(upd.Data))

	event := upd.Data[0]
	expectedEvent := api.Event{
		Type:        "issue",
		Title:       "title",
		Description: strings.Repeat("b", scrapper.MaxDescriptionLength),
		Username:    "name",
		CreatedAt:   time.Unix(0, 0).UTC(),
	}

	s.Equal(*domain.ApiEventToEvent(&expectedEvent), event)
}

func (s *GithubSuite) TestGetUpdateTimeout() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	scr := scrapper.NewGithubScrapper(&config.HTTPConfig{Timeout: s.cfg.Timeout}, &config.CircuitBreakerConfig{}, "test-token", logger)
	scr.ApiScheme = "http"
	scr.ApiHost = s.ts.Listener.Addr().String()
	scr.Client = s.ts.Client()

	updateUrl := "https://github.com" + s.cfg.TimeoutPath
	_, err := scr.GetUpdate(updateUrl)
	s.Error(err)
}

func (s *GithubSuite) TestCircuitBreaker() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	scr := scrapper.NewGithubScrapper(&config.HTTPConfig{Timeout: 50 * time.Millisecond, RetryCount: 1}, s.cb, "test-token", logger)
	scr.ApiScheme = "http"
	scr.ApiHost = s.ts.Listener.Addr().String()

	baseUrl := "https://github.com"
	timeoutUrl := baseUrl + s.cfg.TimeoutPath
	okUrl := baseUrl + s.cfg.OkPath

	// open
	_, err := scr.GetUpdate(timeoutUrl)
	s.True(errors.Is(err, uerrors.ErrAPIUnavailable))
	_, err = scr.GetUpdate(timeoutUrl)
	s.True(errors.Is(err, uerrors.ErrAPIUnavailable))
	_, err = scr.GetUpdate(okUrl)
	s.True(errors.Is(err, uerrors.ErrOpenState))

	// half-open and recover
	time.Sleep(s.cb.WaitDurationInOpenState + 50*time.Millisecond)
	_, err = scr.GetUpdate(okUrl)
	s.NoError(err)
	_, err = scr.GetUpdate(okUrl)
	s.NoError(err)

	// still ok
	_, err = scr.GetUpdate(timeoutUrl)
	s.True(errors.Is(err, uerrors.ErrAPIUnavailable))
	_, err = scr.GetUpdate(okUrl)
	s.NoError(err)

	// open again
	time.Sleep(s.cb.SlidingWindowSize + 50*time.Millisecond)
	_, err = scr.GetUpdate(timeoutUrl)
	s.True(errors.Is(err, uerrors.ErrAPIUnavailable))
	_, err = scr.GetUpdate(timeoutUrl)
	s.True(errors.Is(err, uerrors.ErrAPIUnavailable))
	_, err = scr.GetUpdate(okUrl)
	s.True(errors.Is(err, uerrors.ErrOpenState))

	// half-open and fail
	time.Sleep(s.cb.WaitDurationInOpenState + 50*time.Millisecond)
	_, err = scr.GetUpdate(okUrl)
	s.NoError(err)
	_, err = scr.GetUpdate(timeoutUrl)
	s.True(errors.Is(err, uerrors.ErrAPIUnavailable))
	_, err = scr.GetUpdate(okUrl)
	fmt.Fprintf(os.Stderr, "err: %v\n", err)
	s.True(errors.Is(err, uerrors.ErrOpenState))
}
