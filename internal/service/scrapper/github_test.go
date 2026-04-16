package scrapper

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

func TestGithubScrapper_GetUpdate_NewIssueFormatsEvent(t *testing.T) {
	t.Parallel()

	createdAt := time.Unix(0, 0).UTC()
	body := strings.Repeat("b", maxDescriptionLength+10)
	url := "/acme/project"
	serverUrl := "/repos" + url

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serverUrl:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"description":"test repo"}`))
		case serverUrl + "/pulls":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case serverUrl + "/issues":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{
					"created_at":"` + createdAt.Format(time.RFC3339) + `",
					"title":"title",
					"body":"` + body + `",
					"user":{"login":"name"}
				}
			]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := NewGithubScrapper("", logger)
	s.apiScheme = "http"
	s.apiHost = ts.Listener.Addr().String()
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
		Description: strings.Repeat("b", maxDescriptionLength),
		Username:    "name",
		CreatedAt:   createdAt,
	}

	assert.Equal(t, expectedEvent, event)
}
