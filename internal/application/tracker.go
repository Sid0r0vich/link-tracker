package application

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"

type Tracker interface {
	AddChat(int64) error
	AddLink(int64, domain.Link) error
}
