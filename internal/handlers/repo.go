package handlers

import (
	"errors"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

var ErrChatAlreadyExists = errors.New("chat already exists")
var ErrLinkAlreadyExists = errors.New("link already exists")
var ErrChatNotExists = errors.New("chat not exists")
var ErrLinkNotFound = errors.New("link not found")

type LinkRepository interface {
	AddChat(int64) error
	DeleteChat(int64) error
	GetLinks(int64) ([]domain.LinkWithID, error)
	AddLink(int64, domain.Link) error
	DeleteLink(int64, string) error
}
