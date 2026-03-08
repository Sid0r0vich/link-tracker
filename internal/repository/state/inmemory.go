package state_repository

import (
	"sync"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/uerrors"
)

type InMemoryStateRepo struct {
	*sync.RWMutex
	data map[int64]domain.BotData
}

func NewInMemoryStateRepo() *InMemoryStateRepo {
	return &InMemoryStateRepo{
		RWMutex: &sync.RWMutex{},
		data:    make(map[int64]domain.BotData),
	}
}

func (r *InMemoryStateRepo) GetData(chatID int64) (domain.BotData, error) {
	r.RLock()
	defer r.RUnlock()

	data, ok := r.data[chatID]

	if !ok {
		return nil, uerrors.ErrChatNotExists
	}

	return data, nil
}

func (r *InMemoryStateRepo) SetData(chatID int64, data domain.BotData) error {
	r.Lock()
	defer r.Unlock()

	r.data[chatID] = data

	return nil
}
