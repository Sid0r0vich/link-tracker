package infrastructure

type LinkNoID struct {
	URL     int64
	Tags    []string
	Filters []string
}

type Link struct {
	ID int64
	LinkNoID
}

type ChatRepository interface {
	AddChat(int64) error
	DeleteChat(int64) error
	GetLinks(int64) ([]Link, error)
	AddLink(int64) error
	DeleteLink(int64, string) error
}
