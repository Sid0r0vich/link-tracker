package link

//go:generate go run go.uber.org/mock/mockgen -source=../scrapper/scrapper.go -destination=mocks/mock_scrapper.go -package=mocks

import (
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper"
)

type LinkService interface {
	AddChat(int64) error
	DeleteChat(int64) error
	GetLinks(int64) ([]domain.LinkWithID, error)
	AddLink(int64, domain.Link) (int64, error)
	DeleteLink(int64, string) (*domain.LinkWithID, error)
}

type LinkServiceImpl struct {
	linkRepo repository.LinkRepository
	scrapper scrapper.Scrapper
}

func NewLinkService(repo repository.LinkRepository, scrapper scrapper.Scrapper) *LinkServiceImpl {
	return &LinkServiceImpl{
		linkRepo: repo,
		scrapper: scrapper,
	}
}

func (s *LinkServiceImpl) AddChat(chatID int64) error {
	return s.linkRepo.AddChat(chatID)
}

func (s *LinkServiceImpl) DeleteChat(chatID int64) error {
	return s.linkRepo.DeleteChat(chatID)
}

func (s *LinkServiceImpl) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	return s.linkRepo.GetLinks(chatID)
}

func (s *LinkServiceImpl) AddLink(chatID int64, link domain.Link) (int64, error) {
	update, err := s.scrapper.GetUpdate(link.URL)
	if err != nil {
		return 0, err
	}

	link.UpdatedAt = update.UpdatedAt
	link.UpdatedAt = time.Time{} // тест, сервис возвращает не актуальные, а все обновления

	id, err := s.linkRepo.AddLink(chatID, link)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *LinkServiceImpl) DeleteLink(chatID int64, linkURL string) (*domain.LinkWithID, error) {
	return s.linkRepo.DeleteLink(chatID, linkURL)
}
