package state_repository

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"

type StateRepository interface {
	GetData(int64) (domain.ChatData, error)
	SetData(int64, domain.ChatData) error
}
