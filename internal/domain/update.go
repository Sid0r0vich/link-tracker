package domain

import (
	"time"
)

type Update struct {
	ID        int64
	URL       string
	UpdatedAt time.Time
	Data      []Event
}

type UpdateMessage struct {
	Data      []Event
	Id        int64
	TgChatIds []int64
	Url       string
}
