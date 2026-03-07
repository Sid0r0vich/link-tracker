package application

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type API interface {
	GetState() domain.BotState
	Send(int64, string)
	StartTrack()
	SetTrackLink(string)
	SetTrackTags([]string) error
	SetTrackFilters([]string) error
	AddLink(int64) error
}
