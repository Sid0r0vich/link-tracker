package domain

type UpdateResponse struct {
	ID        int64   `json:"id"`
	URL       string  `json:"url"`
	Desc      string  `json:"description"`
	TgChatIds []int64 `json:"tgChatIds"`
}
