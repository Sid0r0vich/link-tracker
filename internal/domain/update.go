package domain

import "time"

type Update struct {
	ID        int64
	URL       string
	Desc      string
	UpdatedAt time.Time
}
