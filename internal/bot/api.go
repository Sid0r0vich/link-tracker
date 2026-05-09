package bot

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=api.go -destination=mocks/mock.gen.go -package=mocks

type API interface {
	GetData(int64) (domain.ChatData, error)
	SetData(int64, domain.ChatData) error
	Send(int64, string)
	StartTrack(int64) error
	StopTrack(int64)
	SetTrackLink(int64, string) error
	SetUntrackLink(int64, string) error
	SetTrackTags(int64, []string) error
	SetTrackFilters(int64, []string) error
	AddChat(int64) error
	DeleteChat(int64) error
	GetLinks(int64, string) ([]domain.LinkWithID, error)
	AddLink(int64) error
	DeleteLink(int64) error
	LogError(error)
	Wait(int64) error
	CheckUrl(string) error
}
