package scrapper

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper/mocks"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

func TestStackoverflowScrapper_GetUpdate_NewAnswerFormatsEvent(t *testing.T) {
	t.Parallel()

	creationDate := time.Now().Unix()
	lastActivity := time.Now().Unix()
	body := strings.Repeat("b", maxDescriptionLength+10)
	url := "/questions/1"

	ts := mocks.NewMockStackoverflowAPI(t, url, creationDate, lastActivity, body)
	defer ts.Close()

	s := NewStackoverflowScrapper("test-key")
	s.ApiScheme = "http"
	s.ApiHost = ts.Listener.Addr().String()
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
