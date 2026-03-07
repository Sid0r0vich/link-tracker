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
}

type BotSimpleData struct {
	State BotState
}

func (d *BotSimpleData) GetState() BotState {
	return d.State
}

type BotTrackData struct {
	BotSimpleData
	Link Link
}

type BotUntrackData struct {
	BotSimpleData
	URL string
}
