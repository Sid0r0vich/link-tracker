package link

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	repoMock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link/mocks"
	scrapperMock "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper/mocks"
	"go.uber.org/mock/gomock"
)

func TestLinkService_AddChat(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMock.NewMockLinkRepository(ctrl)
	scr := scrapperMock.NewMockScrapper(ctrl)
	service := NewLinkService(repo, scr)
	сhatID := int64(42)

	repo.EXPECT().AddChat(сhatID).Return(nil)

	assert.NoError(t, service.AddChat(сhatID))
}

func TestLinkService_DeleteChat(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMock.NewMockLinkRepository(ctrl)
	scr := scrapperMock.NewMockScrapper(ctrl)
	service := NewLinkService(repo, scr)
	chatID := int64(12)

	repo.EXPECT().AddChat(chatID).Return(nil)
	repo.EXPECT().DeleteChat(chatID).Return(nil)

	assert.NoError(t, service.AddChat(chatID))
	assert.NoError(t, service.DeleteChat(chatID))
}

func TestLinkService_GetLinks(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMock.NewMockLinkRepository(ctrl)
	scr := scrapperMock.NewMockScrapper(ctrl)
	service := NewLinkService(repo, scr)
	chatID := int64(7)

	expected := []domain.LinkWithID{{ID: 1, Link: domain.Link{URL: "https://example.com"}}}
	repo.EXPECT().GetLinks(chatID).Return(expected, nil)

	links, err := service.GetLinks(chatID)
	assert.NoError(t, err)
	assert.Equal(t, expected, links)
}

func TestLinkService_AddLink_ForwardError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMock.NewMockLinkRepository(ctrl)
	scr := scrapperMock.NewMockScrapper(ctrl)
	service := NewLinkService(repo, scr)

	input := domain.Link{URL: "https://bad.example.com"}
	scr.EXPECT().GetUpdate(input.URL).Return(nil, uerrors.ErrBadURL)

	id, err := service.AddLink(1, input)
	if !errors.Is(err, uerrors.ErrBadURL) {
		t.Fatalf("expected ErrBadURL, got id=%d err=%v", id, err)
	}
}

func TestLinkService_AddLink_SetsZeroUpdatedAtBeforeRepository(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMock.NewMockLinkRepository(ctrl)
	scr := scrapperMock.NewMockScrapper(ctrl)
	service := NewLinkService(repo, scr)

	chatID := int64(5)
	linkID := int64(99)
	updatedAt := time.Unix(0, 0).UTC()
	input := domain.Link{URL: "https://example.com", LinkInfo: domain.LinkInfo{Tags: []string{"go"}}}

	scr.EXPECT().GetUpdate(input.URL).Return(&domain.Update{UpdatedAt: updatedAt}, nil)
	repo.EXPECT().AddLink(chatID, gomock.Any()).DoAndReturn(func(_ int64, got domain.Link) (int64, error) {
		if !got.UpdatedAt.IsZero() {
			t.Fatalf("expected zero UpdatedAt, got %v", got.UpdatedAt)
		}
		if got.URL != input.URL {
			t.Fatalf("unexpected URL: %q", got.URL)
		}
		if len(got.Tags) != 1 || got.Tags[0] != "go" {
			t.Fatalf("unexpected tags: %+v", got.Tags)
		}
		return linkID, nil
	})

	id, err := service.AddLink(chatID, input)
	assert.NoError(t, err)
	assert.Equal(t, linkID, id)
}

func TestLinkService_DeleteLink(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMock.NewMockLinkRepository(ctrl)
	scr := scrapperMock.NewMockScrapper(ctrl)
	service := NewLinkService(repo, scr)
	url := "https://example.com"
	chatID := int64(422)
	expectedLink := domain.LinkWithID{
		ID: int64(1),
		Link: domain.Link{
			URL: url,
		},
	}

	repo.EXPECT().DeleteLink(chatID, url).Return(&expectedLink, nil)

	gotLink, err := service.DeleteLink(chatID, url)
	assert.NoError(t, err)
	assert.Equal(t, expectedLink, *gotLink)
}
