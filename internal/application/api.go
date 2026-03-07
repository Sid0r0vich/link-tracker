package application

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type API interface {
	GetState() domain.BotState
	Send(int64, string)
	StartTrack()
	StopTrack()
	SetTrackLink(string)
	SetUntrackLink(string)
	SetTrackTags([]string) error
	SetTrackFilters([]string) error
	AddChat(int64) error
	DeleteChat(int64) error
	GetLinks(int64) ([]domain.LinkWithID, error)
	AddLink(int64) error
	DeleteLink(int64) error
	LogError(error)
}
