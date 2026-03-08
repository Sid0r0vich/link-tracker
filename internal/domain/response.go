package domain

type LinkResponse struct {
	ID      int64    `json:"id"`
	URL     string   `json:"url"`
	Tags    []string `json:"tags"`
	Filters []string `json:"filters"`
}

type LinksResponse struct {
	Links []LinkResponse `json:"links"`
	Size  int            `json:"size"`
}

type ErrorResponse struct {
	Description      string   `json:"description"`
	Code             string   `json:"code"`
	ExceptionName    string   `json:"exceptionName"`
	ExceptionMessage string   `json:"exceptionMessage"`
	Stacktrace       []string `json:"stacktrace"`
}

type UpdateResponse struct {
	ID        int64   `json:"id"`
	URL       string  `json:"url"`
	Desc      string  `json:"description"`
	TgChatIds []int64 `json:"tgChatIds"`
}
