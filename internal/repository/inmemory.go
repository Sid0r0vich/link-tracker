package repository

import (
	"sync"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/uerrors"
)

type InMemoryLinkRepo struct {
	*sync.RWMutex
	data map[int64]map[string]domain.LinkInfoWithID
	size int64
}

func NewInMemoryLinkRepo() *InMemoryLinkRepo {
	return &InMemoryLinkRepo{
		RWMutex: &sync.RWMutex{},
		data:    make(map[int64]map[string]domain.LinkInfoWithID),
	}
}

func (r *InMemoryLinkRepo) AddChat(chatID int64) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.data[chatID]; ok {
		return uerrors.ErrChatAlreadyExists
	}

	r.data[chatID] = make(map[string]domain.LinkInfoWithID, 0)

	return nil
}

func (r *InMemoryLinkRepo) DeleteChat(chatID int64) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.data[chatID]; !ok {
		return uerrors.ErrChatNotExists
	}

	delete(r.data, chatID)

	return nil
}

func (r *InMemoryLinkRepo) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	r.RLock()
	defer r.RUnlock()

	chat, ok := r.data[chatID]
	if !ok {
		return []domain.LinkWithID{}, uerrors.ErrChatNotExists
	}

	links := make([]domain.LinkWithID, len(chat))
	cnt := 0
	for url, link := range chat {
		links[cnt] = domain.LinkWithID{Link: domain.Link{URL: url, LinkInfo: link.LinkInfo}, ID: link.ID}
		cnt++
	}

	return links, nil
}

func (r *InMemoryLinkRepo) AddLink(chatID int64, link domain.Link) (int64, error) {
	r.Lock()
	defer r.Unlock()

	chat, ok := r.data[chatID]
	if !ok {
		return 0, uerrors.ErrChatNotExists
	}

	if _, ok = chat[link.URL]; ok {
		return 0, uerrors.ErrLinkAlreadyExists
	}

	chat[link.URL] = domain.LinkInfoWithID{LinkInfo: link.LinkInfo, ID: r.size}
	r.size++

	return r.size - 1, nil
}

func (r *InMemoryLinkRepo) DeleteLink(chatID int64, url string) (*domain.LinkWithID, error) {
	r.Lock()
	defer r.Unlock()

	chat, ok := r.data[chatID]
	if !ok {
		return nil, uerrors.ErrChatNotExists
	}

	link, ok := chat[url]
	if !ok {
		return nil, uerrors.ErrLinkNotFound
	}

	delete(chat, url)

	return &domain.LinkWithID{
		Link: domain.Link{
			LinkInfo: domain.LinkInfo{
				Tags:    link.Tags,
				Filters: link.Filters,
			},
			URL: url,
		},
		ID: link.ID,
	}, nil
}
