package domain

type ChatState int

const (
	Wait ChatState = iota
	LinkTrack
	TagsTrack
	FilterTrack
	LinkUntrack
)

type ChatData interface {
	GetState() ChatState
	SetState(ChatState)
}

type ChatSimpleData struct {
	State ChatState
}

func (d ChatSimpleData) GetState() ChatState {
	return d.State
}

func (d ChatSimpleData) SetState(s ChatState) {
	d.State = s
}

type ChatTrackData struct {
	ChatSimpleData
	Link Link
}

type ChatUntrackData struct {
	ChatSimpleData
	URL string
}
