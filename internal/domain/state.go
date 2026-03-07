package domain

type BotState int

const (
	Wait BotState = iota
	StartTrack
	LinkTrack
	TagTrack
)
