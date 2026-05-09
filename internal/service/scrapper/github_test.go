package scrapper_test

import (
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper/mocks"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

func TestGithubScrapper_GetUpdate_NewIssueFormatsEvent(t *testing.T) {
	t.Parallel()

	createdAt := time.Unix(0, 0).UTC()
	body := strings.Repeat("b", scrapper.MaxDescriptionLength+10)
	url := "/acme/project"
	serverUrl := "/repos" + url

	ts := mocks.NewMockGithubAPI(t, serverUrl, createdAt, body)
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := scrapper.NewGithubScrapper(&config.GithubConfig{}, logger)
	s.ApiScheme = "http"
	s.ApiHost = ts.Listener.Addr().String()
	s.Client = *ts.Client()

	updateUrl := "https://github.com" + url
	upd, err := s.GetUpdate(updateUrl)
	assert.NoError(t, err)
	assert.NotNil(t, upd)
	assert.Equal(t, updateUrl, upd.URL)
	assert.Equal(t, 1, len(upd.Data))

	event := upd.Data[0]
	expectedEvent := api.Event{
		Type:        "issue",
		Title:       "title",
		Description: strings.Repeat("b", scrapper.MaxDescriptionLength),
		Username:    "name",
		CreatedAt:   createdAt,
	}

	assert.Equal(t, *domain.ApiEventToEvent(&expectedEvent), event)
}
