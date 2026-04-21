package scheduler_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	repoMock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link/mocks"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scheduler"
	scrapperMock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper/mocks"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/update"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
	"go.uber.org/mock/gomock"
)

func TestScheduler_CheckUpdates_PartialFailuresAreIsolated(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	var received []api.UpdateResponse

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/updates" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var payload api.UpdateResponse
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		received = append(received, payload)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	updater, err := update.NewUpdateRestService(server.URL)
	assert.NoError(t, err)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := repoMock.NewMockLinkUpdateRepository(ctrl)
	scr := scrapperMock.NewMockScrapper(ctrl)

	eventTime := time.Unix(0, 0).UTC()
	okUrl := "https://example.com/ok"
	failUrl := "https://example.com/fail"
	batchSize := int64(2)
	okChatIDs := []int64{2, 3}

	repo.EXPECT().GetLinkBatch(int64(0)).Return([]domain.LinkUpdate{
		{URL: failUrl, IDs: []int64{1}},
		{URL: okUrl, IDs: okChatIDs},
	}, batchSize, nil)

	scr.EXPECT().GetUpdate(failUrl).Return(nil, errors.New("upstream unavailable"))

	event := domain.Event{
		Type:        "issue",
		Title:       "title",
		Username:    "name",
		Description: "description",
		CreatedAt:   eventTime,
	}
	upd := domain.Update{
		ID:        7,
		URL:       okUrl,
		UpdatedAt: eventTime,
		Data:      []domain.Event{event},
	}
	scr.EXPECT().GetUpdate(okUrl).Return(&upd, nil)

	repo.EXPECT().GetTimeAndUpdateLink(okUrl, eventTime).Return(time.Time{}, nil)
	repo.EXPECT().GetLinkBatch(batchSize).Return([]domain.LinkUpdate{}, batchSize, nil)

	s, err := scheduler.NewScheduler(repo, logger, updater, scr, time.Second)
	if err != nil {
		t.Fatalf("NewScheduler returned error: %v", err)
	}

	if err := s.CheckUpdates(); err != nil {
		t.Fatalf("checkUpdates: %v", err)
	}

	assert.Equal(t, 1, len(received))
	gotUpdResp := received[0]

	assert.Equal(t, okUrl, gotUpdResp.Url)
	assert.Equal(t, okChatIDs, gotUpdResp.TgChatIds)
	assert.Equal(t, 1, len(gotUpdResp.Data))
	assert.Equal(t, event, *domain.ApiEventToEvent(&gotUpdResp.Data[0]))
}
