package domain

import "time"

type LinkInfo struct {
	Tags      []string `json:"tags"`
	Filters   []string `json:"filters"`
	UpdatedAt time.Time
}

type LinkInfoWithID struct {
	LinkInfo
	ID int64
}

type Link struct {
	LinkInfo
	URL string `json:"link"`
}

type LinkWithID struct {
	Link
	ID int64 `json:"id"`
}

type LinkUpdate struct {
	IDs       map[int64]struct{}
	UpdatedAt time.Time
}
