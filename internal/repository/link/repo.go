package link_repository

import (
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=repo.go -destination=mocks/mock.gen.go -package=mocks

type LinkRepository interface {
	AddChat(int64) error
	DeleteChat(int64) error
	GetLinks(int64) ([]domain.LinkWithID, error)
	AddLink(int64, domain.Link) (int64, error)
	DeleteLink(int64, string) (*domain.LinkWithID, error)
}

type LinkUpdateRepository interface {
	GetTimeAndUpdateLink(string, time.Time) (time.Time, error)
	GetAllLinks() ([]domain.LinkUpdate, error)
}

type LinkUnitedRepository interface {
	LinkRepository
	LinkUpdateRepository
}
