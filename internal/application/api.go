package application

import (
	"errors"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

var ErrBotAlreadyWaiting = errors.New("bot already waiting")

type API interface {
	GetData(int64) (domain.BotData, error)
	SetData(int64, domain.BotData) error
	Send(int64, string)
	StartTrack(int64) error
	StopTrack(int64)
	SetTrackLink(int64, string) error
	SetUntrackLink(int64, string) error
	SetTrackTags(int64, []string) error
	SetTrackFilters(int64, []string) error
	AddChat(int64) error
	DeleteChat(int64) error
	GetLinks(int64) ([]domain.LinkWithID, error)
	AddLink(int64) error
	DeleteLink(int64) error
	LogError(error)
	Wait(int64) error
}
