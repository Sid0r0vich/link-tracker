package scrapper

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

func TestStackoverflowScrapper_GetUpdate_NewAnswerFormatsEvent(t *testing.T) {
	t.Parallel()

	creationDate := time.Now().Unix()
	lastActivity := time.Now().Unix()
	body := strings.Repeat("b", maxDescriptionLength+10)
	url := "/questions/1"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case url:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"items":[
					{"last_activity_date":` + int64ToString(lastActivity) + `,"title":"title"}
				]
			}`))
		case url + "/answers":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"items":[
					{
						"last_activity_date":` + int64ToString(lastActivity) + `,
						"creation_date":` + int64ToString(creationDate) + `,
						"owner":{"display_name":"name"},
						"body":"` + body + `"
					}
				]
			}`))
		case url + "/comments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	s := NewStackoverflowScrapper("test-key")
	s.apiScheme = "http"
	s.apiHost = ts.Listener.Addr().String()
	s.Client = *ts.Client()

	updateUrl := "https://stackoverflow.com" + url
	upd, err := s.GetUpdate(updateUrl)
	assert.NoError(t, err)
	assert.NotNil(t, upd)
	assert.Equal(t, updateUrl, upd.URL)
	assert.Equal(t, 1, len(upd.Data))

	event := upd.Data[0]
	expectedEvent := api.Event{
		Type:        "answer",
		Title:       "title",
		Description: strings.Repeat("b", maxDescriptionLength),
		Username:    "name",
		CreatedAt:   time.Unix(creationDate, 0),
	}

	assert.Equal(t, *domain.ApiEventToEvent(&expectedEvent), event)
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
