package scrapper_repository

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"

type Scrapper interface {
	GetUpdate(string) (*domain.Update, error)
}
