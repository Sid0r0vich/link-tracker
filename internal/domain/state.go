package domain

type BotState int

const (
	Wait BotState = iota
	LinkTrack
	TagsTrack
	FilterTrack
	LinkUntrack
)

type BotData interface {
	GetState() BotState
	SetState(BotState)
}

type BotSimpleData struct {
	State BotState
}

func (d BotSimpleData) GetState() BotState {
	return d.State
}

func (d BotSimpleData) SetState(s BotState) {
	d.State = s
}

type BotTrackData struct {
	BotSimpleData
	Link Link
}

type BotUntrackData struct {
	BotSimpleData
	URL string
}
