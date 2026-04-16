package repository_test

import (
	"errors"
	"sort"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	link_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
)

const (
	SubscriptionBatchSize = 2
)

func LinkRepo_AddAndDeleteChatTest(t *testing.T, repo link_repository.LinkUnitedRepository) {
	assert.NoError(t, repo.AddChat(1))

	if err := repo.AddChat(1); !errors.Is(err, uerrors.ErrChatAlreadyExists) {
		t.Fatalf("expected ErrChatAlreadyExists, got: %v", err)
	}

	if err := repo.DeleteChat(2); !errors.Is(err, uerrors.ErrChatNotExists) {
		t.Fatalf("expected ErrChatNotExists, got: %v", err)
	}

	assert.NoError(t, repo.DeleteChat(1))
}

func LinkRepo_GetLinksChatNotExistsTest(t *testing.T, repo link_repository.LinkUnitedRepository) {
	_, err := repo.GetLinks(42)
	if !errors.Is(err, uerrors.ErrChatNotExists) {
		t.Fatalf("expected ErrChatNotExists, got: %v", err)
	}
}

func LinkRepo_AddLinkAndGetLinksTest(t *testing.T, repo link_repository.LinkUnitedRepository) {
	assert.NoError(t, repo.AddChat(1))

	linkTime := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	chatSubscriptionID, err := repo.AddLink(1, domain.Link{
		URL: "https://example.com/a",
		LinkInfo: domain.LinkInfo{
			Tags:      []string{"go", "backend"},
			UpdatedAt: linkTime,
		},
	})
	assert.NoError(t, err)

	links, err := repo.GetLinks(1)
	assert.NoError(t, err)

	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}

	got := links[0]
	if got.ID != chatSubscriptionID {
		t.Fatalf("unexpected chat subscription id: got %d, want %d", got.ID, chatSubscriptionID)
	}
	if got.URL != "https://example.com/a" {
		t.Fatalf("unexpected url: %s", got.URL)
	}
	if !got.UpdatedAt.Equal(linkTime) {
		t.Fatalf("unexpected updwated_at: got %v, want %v", got.UpdatedAt, linkTime)
	}

	sort.Strings(got.Tags)
	wantTags := []string{"backend", "go"}
	assert.Equal(t, wantTags, got.Tags)
}

func LinkRepo_AddLinkChatNotExistsTest(t *testing.T, repo link_repository.LinkUnitedRepository) {
	_, err := repo.AddLink(100, domain.Link{URL: "https://example.com", LinkInfo: domain.LinkInfo{UpdatedAt: time.Now()}})
	if !errors.Is(err, uerrors.ErrChatNotExists) {
		t.Fatalf("expected ErrChatNotExists, got: %v", err)
	}
}

func LinkRepo_AddLinkAlreadyExistsTest(t *testing.T, repo link_repository.LinkUnitedRepository) {
	chatID := int64(7)
	link := domain.Link{URL: "https://example.com/duplicate", LinkInfo: domain.LinkInfo{UpdatedAt: time.Now()}}

	assert.NoError(t, repo.AddChat(chatID))

	_, err := repo.AddLink(chatID, link)
	assert.NoError(t, err)

	if _, err := repo.AddLink(chatID, link); !errors.Is(err, uerrors.ErrLinkAlreadyExists) {
		t.Fatalf("expected ErrLinkAlreadyExists, got: %v", err)
	}
}

func LinkRepo_DeleteLinkTest(t *testing.T, repo link_repository.LinkUnitedRepository) {
	chatID := int64(42)
	assert.NoError(t, repo.AddChat(chatID))

	linkID, err := repo.AddLink(chatID, domain.Link{URL: "https://example.com/x", LinkInfo: domain.LinkInfo{UpdatedAt: time.Now()}})
	assert.NoError(t, err)

	if _, err := repo.DeleteLink(chatID, "https://example.com/missing"); !errors.Is(err, uerrors.ErrLinkNotFound) {
		t.Fatalf("expected ErrLinkNotFound, got: %v", err)
	}

	deleted, err := repo.DeleteLink(chatID, "https://example.com/x")
	assert.NoError(t, err)
	if deleted == nil || deleted.ID != linkID {
		t.Fatalf("expected deleted link id == %d, got: %d", linkID, deleted.ID)
	}

	links, err := repo.GetLinks(chatID)
	assert.NoError(t, err)
	if len(links) != 0 {
		t.Fatalf("expected no links after delete, got %d", len(links))
	}

	if _, err := repo.DeleteLink(999, "https://example.com/x"); !errors.Is(err, uerrors.ErrChatNotExists) {
		t.Fatalf("expected ErrChatNotExists, got: %v", err)
	}
}

func LinkRepo_GetTimeAndUpdateLinkTest(t *testing.T, repo link_repository.LinkUnitedRepository) {
	chatID := int64(2)
	assert.NoError(t, repo.AddChat(chatID))

	original := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	_, err := repo.AddLink(chatID, domain.Link{URL: "https://example.com/time", LinkInfo: domain.LinkInfo{UpdatedAt: original}})
	assert.NoError(t, err)

	prev, err := repo.GetTimeAndUpdateLink("https://example.com/time", original.Add(-time.Hour))
	assert.NoError(t, err)
	if !prev.Equal(original) {
		t.Fatalf("expected previous time to be %v, got %v", original, prev)
	}

	newer := original.Add(2 * time.Hour)
	prev, err = repo.GetTimeAndUpdateLink("https://example.com/time", newer)
	assert.NoError(t, err)
	if !prev.Equal(original) {
		t.Fatalf("expected previous time to be %v, got %v", original, prev)
	}

	links, err := repo.GetLinks(chatID)
	assert.NoError(t, err)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if !links[0].UpdatedAt.Equal(newer) {
		t.Fatalf("expected updated_at to be %v, got %v", newer, links[0].UpdatedAt)
	}
}

func LinkRepo_GetTimeAndUpdateLinkNotFoundTest(t *testing.T, repo link_repository.LinkUnitedRepository) {
	chatID := int64(33)
	link := "https://example.com/new/link42"
	assert.NoError(t, repo.AddChat(chatID))

	start := time.Now()
	_, err := repo.AddLink(chatID, domain.Link{URL: link, LinkInfo: domain.LinkInfo{UpdatedAt: time.Now()}})
	assert.NoError(t, err)

	returned, err := repo.GetTimeAndUpdateLink(link, start)
	end := time.Now()
	assert.NoError(t, err)
	if returned.Before(start) || returned.After(end) {
		t.Fatalf("expected returned time around now, got %v, start: %v, end: %v", returned, start, end)
	}
}

func LinkRepo_GetLinkBatchTest(t *testing.T, repo link_repository.LinkUnitedRepository) {
	firChat := int64(3)
	secChat := int64(4)
	linkA := "https://example.com/aa"
	linkB := "https://example.com/bb"
	linkC := "https://example.com/cc"

	assert.NoError(t, repo.AddChat(firChat))
	assert.NoError(t, repo.AddChat(secChat))
	assert.Error(t, repo.AddChat(secChat))

	t0 := time.Unix(0, 0).UTC()
	_, err := repo.AddLink(firChat, domain.Link{URL: linkA, LinkInfo: domain.LinkInfo{UpdatedAt: t0}})
	assert.NoError(t, err)

	_, err = repo.AddLink(secChat, domain.Link{URL: linkA, LinkInfo: domain.LinkInfo{UpdatedAt: t0}})
	assert.NoError(t, err)

	_, err = repo.AddLink(firChat, domain.Link{URL: linkB, LinkInfo: domain.LinkInfo{UpdatedAt: t0.Add(time.Hour)}})
	assert.NoError(t, err)

	_, err = repo.AddLink(firChat, domain.Link{URL: linkC, LinkInfo: domain.LinkInfo{UpdatedAt: t0.Add(2 * time.Hour)}})
	assert.NoError(t, err)

	batch, lastID, err := repo.GetLinkBatch(0)
	assert.NoError(t, err)
	if len(batch) != 2 {
		t.Fatalf("expected first batch size 2, got %d", len(batch))
	}
	if lastID <= 0 {
		t.Fatalf("expected last id > 0, got %d", lastID)
	}

	urlToIDs := make(map[string][]int64, len(batch))
	for _, upd := range batch {
		sort.Slice(upd.IDs, func(i, j int) bool { return upd.IDs[i] < upd.IDs[j] })
		urlToIDs[upd.URL] = upd.IDs
	}

	idsForA, ok := urlToIDs[linkA]
	if !ok {
		t.Fatalf("expected url aa in first batch, got %#v", batch)
	}
	assert.Equal(t, []int64{firChat, secChat}, idsForA)

	if _, ok := urlToIDs[linkB]; !ok {
		t.Fatalf("expected url bb in first batch, got %#v", batch)
	}

	nextBatch, nextLastID, err := repo.GetLinkBatch(lastID)
	assert.NoError(t, err)
	if len(nextBatch) != 1 {
		t.Fatalf("expected second batch size 1, got %d", len(nextBatch))
	}
	if nextBatch[0].URL != linkC {
		t.Fatalf("expected url cc in second batch, got %s", nextBatch[0].URL)
	}
	if nextLastID <= lastID {
		t.Fatalf("expected next last id > first last id, got %d <= %d", nextLastID, lastID)
	}
}
