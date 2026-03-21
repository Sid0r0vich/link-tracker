package link_repository

import (
	"sync"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/uerrors"
)

type InMemoryLinkRepo struct {
	*sync.RWMutex
	urlToChatLinks  map[int64]map[string]domain.LinkInfoWithID
	size            int64
	urlToLinkUpdate map[string]domain.LinkUpdate
}

func NewInMemoryLinkRepo() *InMemoryLinkRepo {
	return &InMemoryLinkRepo{
		RWMutex:         &sync.RWMutex{},
		urlToChatLinks:  make(map[int64]map[string]domain.LinkInfoWithID),
		urlToLinkUpdate: make(map[string]domain.LinkUpdate),
	}
}

func (r *InMemoryLinkRepo) AddChat(chatID int64) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.urlToChatLinks[chatID]; ok {
		return uerrors.ErrChatAlreadyExists
	}

	r.urlToChatLinks[chatID] = make(map[string]domain.LinkInfoWithID, 0)

	return nil
}

func (r *InMemoryLinkRepo) DeleteChat(chatID int64) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.urlToChatLinks[chatID]; !ok {
		return uerrors.ErrChatNotExists
	}

	delete(r.urlToChatLinks, chatID)

	return nil
}

func (r *InMemoryLinkRepo) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	r.RLock()
	defer r.RUnlock()

	chat, ok := r.urlToChatLinks[chatID]
	if !ok {
		return []domain.LinkWithID{}, uerrors.ErrChatNotExists
	}

	links := make([]domain.LinkWithID, len(chat))
	cnt := 0
	for url, link := range chat {
		links[cnt] = *domain.LinkInfoWithIDToLinkWithID(&link, url)
		cnt++
	}

	return links, nil
}

func (r *InMemoryLinkRepo) AddLink(chatID int64, link domain.Link) (int64, error) {
	r.Lock()
	defer r.Unlock()

	chat, ok := r.urlToChatLinks[chatID]
	if !ok {
		return 0, uerrors.ErrChatNotExists
	}

	if _, ok = chat[link.URL]; ok {
		return 0, uerrors.ErrLinkAlreadyExists
	}

	chat[link.URL] = domain.LinkInfoWithID{LinkInfo: link.LinkInfo, ID: r.size}
	r.size++

	linkUpd, ok := r.urlToLinkUpdate[link.URL]
	if !ok {
		linkUpd = domain.LinkUpdate{UpdatedAt: time.Now(), IDs: make(map[int64]struct{})}
	}

	linkUpd.IDs[chatID] = struct{}{}
	r.urlToLinkUpdate[link.URL] = linkUpd

	return r.size - 1, nil
}

func (r *InMemoryLinkRepo) DeleteLink(chatID int64, url string) (*domain.LinkWithID, error) {
	r.Lock()
	defer r.Unlock()

	chat, ok := r.urlToChatLinks[chatID]
	if !ok {
		return nil, uerrors.ErrChatNotExists
	}

	link, ok := chat[url]
	if !ok {
		return nil, uerrors.ErrLinkNotFound
	}

	delete(chat, url)

	linkUpd, ok := r.urlToLinkUpdate[url]
	if ok {
		delete(linkUpd.IDs, chatID)
		r.urlToLinkUpdate[url] = linkUpd
	}

	return domain.LinkInfoWithIDToLinkWithID(&link, url), nil
}

func (r *InMemoryLinkRepo) CheckLinkExists(chatID int64, url string) (bool, error) {
	r.RLock()
	defer r.RUnlock()

	chat, ok := r.urlToChatLinks[chatID]
	if !ok {
		return false, nil
	}

	_, ok = chat[url]
	return ok, nil
}

func (r *InMemoryLinkRepo) GetAllLinks() (map[string]domain.LinkUpdate, error) {
	r.RLock()
	defer r.RUnlock()

	return r.urlToLinkUpdate, nil
}

func (r *InMemoryLinkRepo) GetTimeAndUpdateLink(url string, updatedAt time.Time) (bool, error) {
	r.Lock()
	defer r.Unlock()

	link := r.urlToLinkUpdate[url]
	if !link.UpdatedAt.Before(updatedAt) {
		return false, nil
	}

	link.UpdatedAt = updatedAt
	r.urlToLinkUpdate[url] = link

	return true, nil
}
