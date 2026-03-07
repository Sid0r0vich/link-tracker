package domain

type BotState int

const (
	Wait BotState = iota
	StartTrack
	LinkTrack
	TagTrack
	FilterTrack
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

func (d *BotTrackData) GetState() BotState {
	return d.BotSimpleData.GetState()
}
