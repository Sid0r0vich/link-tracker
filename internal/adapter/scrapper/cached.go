package scrapper

import (
	"encoding/json"
	"errors"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/cache"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type CachedScrapperAdapter struct {
	base   ScrapperAdapter
	cache  cache.Cache
	logger *slog.Logger
}

func NewCachedScrapperAdapter(base ScrapperAdapter, c cache.Cache, logger *slog.Logger) *CachedScrapperAdapter {
	return &CachedScrapperAdapter{base: base, cache: c, logger: logger}
}

func (a *CachedScrapperAdapter) AddChat(chatID int64) error {
	return a.base.AddChat(chatID)
}

func (a *CachedScrapperAdapter) DeleteChat(chatID int64) error {
	return a.base.DeleteChat(chatID)
}

func (a *CachedScrapperAdapter) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	if cached, err := a.cache.Get(chatID); err == nil {
		a.logger.Debug("cache hit", "chatID", chatID)

		var links []domain.LinkWithID
		if uerr := json.Unmarshal(cached, &links); uerr == nil {
			return links, nil
		}
		if cacheErr := a.cache.Delete(chatID); cacheErr != nil {
			a.logger.Error("failed to delete cache after GetLinks", "chatID", chatID, "error", cacheErr)
		}
	} else if errors.Is(err, cache.ErrCacheMiss) {
		a.logger.Debug("cache miss", "chatID", chatID)
	}

	links, err := a.base.GetLinks(chatID)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(links); err == nil {
		if cacheErr := a.cache.Set(chatID, data); cacheErr != nil {
			a.logger.Error("failed to set cache after GetLinks", "chatID", chatID, "error", cacheErr)
		}
	}

	return links, nil
}

func (a *CachedScrapperAdapter) AddLink(chatID int64, link domain.Link) error {
	return a.base.AddLink(chatID, link)
}

func (a *CachedScrapperAdapter) DeleteLink(chatID int64, url string) error {
	return a.base.DeleteLink(chatID, url)
}
