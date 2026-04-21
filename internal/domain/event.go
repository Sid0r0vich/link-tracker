package domain

import "time"

type Event struct {
	CreatedAt   time.Time
	Description string
	Title       string
	Type        string
	Username    string
}
