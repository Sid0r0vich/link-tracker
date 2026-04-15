package domain

import (
	"time"

	"github.com/lib/pq"
)

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
	IDs       []int64
	URL       string
	UpdatedAt time.Time
}

type DbLink struct {
	ID        int64          `db:"id"`
	Tags      pq.StringArray `db:"tags"`
	UpdatedAt time.Time      `db:"updated_at"`
	URL       string         `db:"url"`
}

type Updatess struct {
	ID int64
}
