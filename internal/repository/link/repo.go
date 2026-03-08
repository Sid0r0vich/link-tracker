package link_repository

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type LinkRepository interface {
	AddChat(int64) error
	DeleteChat(int64) error
	GetLinks(int64) ([]domain.LinkWithID, error)
	AddLink(int64, domain.Link) (int64, error)
	DeleteLink(int64, string) (*domain.LinkWithID, error)
}
