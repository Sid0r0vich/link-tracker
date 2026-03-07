package domain

type LinkInfo struct {
	Tags    []string `json:"tags"`
	Filters []string `json:"filters"`
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
