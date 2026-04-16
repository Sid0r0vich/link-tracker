package domain

import (
	"time"

	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

type Update struct {
	ID        int64
	URL       string
	UpdatedAt time.Time
	Data      []api.Event
}
