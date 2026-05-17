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

type StackoverflowSuite struct {
	suite.Suite
	ts           *httptest.Server
	cfg          *mocks.ApiConfig
	cb           *config.CircuitBreakerConfig
	creationDate int64
	lastActivity int64
}

func (s *StackoverflowSuite) SetupTest() {
	s.creationDate = time.Now().Unix()
	s.lastActivity = time.Now().Unix()
	body := strings.Repeat("b", scrapper.MaxDescriptionLength+10)

	s.cfg = &mocks.ApiConfig{
		ServerUrl:   "/questions",
		OkPath:      "/1",
		TimeoutPath: "/timeout",
		FailPath:    "/fail",
		Body:        body,
		Timeout:     200 * time.Millisecond,
	}
	s.cb = &config.CircuitBreakerConfig{
		SlidingWindowSize:        500 * time.Millisecond,
		SlidingWindowBucketSize:  100 * time.Millisecond,
		MinimumRequiredCalls:     2,
		FailureRateThreshold:     51,
		PermittedCallsInHalfOpen: 4,
		WaitDurationInOpenState:  100 * time.Millisecond,
	}
	s.ts = mocks.NewMockStackoverflowAPI(s.T(), s.cfg, s.creationDate, s.lastActivity)
}

func (s *StackoverflowSuite) TearDownTest() {
	s.ts.Close()
}

func TestStackoverflowSuite(t *testing.T) {
	suite.Run(t, new(StackoverflowSuite))
}

func (s *StackoverflowSuite) TestGetUpdate_NewAnswerFormatsEvent() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cb := &config.CircuitBreakerConfig{
		SlidingWindowSize:        10,
		MinimumRequiredCalls:     1,
		FailureRateThreshold:     100,
		PermittedCallsInHalfOpen: 1,
		WaitDurationInOpenState:  1 * time.Second,
	}
	scr := scrapper.NewStackoverflowScrapper(&config.HTTPConfig{Timeout: 5 * time.Second}, cb, "test-key", logger)
	scr.ApiScheme = "http"
	scr.ApiHost = s.ts.Listener.Addr().String()
	scr.Client = s.ts.Client()

	updateUrl := "https://stackoverflow.com/questions/1"
	upd, err := scr.GetUpdate(updateUrl)
	s.NoError(err)
	s.NotNil(upd)
	s.Equal(updateUrl, upd.URL)
	s.Equal(1, len(upd.Data))

	event := upd.Data[0]
	expectedEvent := api.Event{
		Type:        "answer",
		Title:       "title",
		Description: strings.Repeat("b", scrapper.MaxDescriptionLength),
		Username:    "name",
		CreatedAt:   time.Unix(s.creationDate, 0),
	}

	s.Equal(*domain.ApiEventToEvent(&expectedEvent), event)
}

func (s *StackoverflowSuite) TestGetUpdate_Timeout() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cb := &config.CircuitBreakerConfig{
		SlidingWindowSize:        10,
		MinimumRequiredCalls:     1,
		FailureRateThreshold:     100,
		PermittedCallsInHalfOpen: 1,
		WaitDurationInOpenState:  1 * time.Second,
	}
	scr := scrapper.NewStackoverflowScrapper(&config.HTTPConfig{Timeout: s.cfg.Timeout}, cb, "test-key", logger)
	scr.ApiScheme = "http"
	scr.ApiHost = s.ts.Listener.Addr().String()
	scr.Client = s.ts.Client()

	updateUrl := "https://stackoverflow.com/questions/timeout"
	_, err := scr.GetUpdate(updateUrl)
	s.Error(err)
}

func (s *StackoverflowSuite) TestCircuitBreaker() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	scr := scrapper.NewStackoverflowScrapper(&config.HTTPConfig{Timeout: 50 * time.Millisecond, RetryCount: 1}, s.cb, "test-key", logger)
	scr.ApiScheme = "http"
	scr.ApiHost = s.ts.Listener.Addr().String()

	baseUrl := "https://stackoverflow.com/questions"
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
