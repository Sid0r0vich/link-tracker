package link

//go:generate go run go.uber.org/mock/mockgen -source=../scrapper/scrapper.go -destination=mocks/mock_scrapper.go -package=mocks

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/cache"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
)

type LinkService interface {
	AddChat(int64) error
	DeleteChat(int64) error
	GetLinks(int64) ([]domain.LinkWithID, error)
	AddLink(int64, domain.Link) (int64, error)
	DeleteLink(int64, string) (*domain.LinkWithID, error)
}

type LinkServiceImpl struct {
	linkRepo               repository.LinkRepository
	scrapper               scrapper.Scrapper
	CheckUrl               func(string) error
	clientCacheInvalidator cache.Invalidator
	logger                 *slog.Logger
}

func NewLinkService(
	repo repository.LinkRepository,
	scrapper scrapper.Scrapper,
	clientCacheInvalidator cache.Invalidator,
	logger *slog.Logger,
) *LinkServiceImpl {
	return &LinkServiceImpl{
		linkRepo:               repo,
		scrapper:               scrapper,
		CheckUrl:               utils.CheckUrl,
		clientCacheInvalidator: clientCacheInvalidator,
		logger:                 logger,
	}
}

func (s *LinkServiceImpl) AddChat(chatID int64) error {
	return s.linkRepo.AddChat(chatID)
}

func (s *LinkServiceImpl) DeleteChat(chatID int64) error {
	if err := s.clientCacheInvalidator.Invalidate(chatID); err != nil {
		s.logger.Error(fmt.Sprintf("delete chat: invalidate cache for chatID %d: %v", chatID, err))
	}

	return s.linkRepo.DeleteChat(chatID)
}

func (s *LinkServiceImpl) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	return s.linkRepo.GetLinks(chatID)
}

func (s *LinkServiceImpl) AddLink(chatID int64, link domain.Link) (int64, error) {
	if err := s.CheckUrl(link.URL); err != nil {
		fmt.Fprint(os.Stderr, "BAD URL!")
		return 0, uerrors.ErrBadURL
	}

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

	if err := s.clientCacheInvalidator.Invalidate(chatID); err != nil {
		s.logger.Error(fmt.Sprintf("add link: invalidate cache for chatID %d: %v", chatID, err))
	}

	return id, nil
}

func (s *LinkServiceImpl) DeleteLink(chatID int64, linkURL string) (*domain.LinkWithID, error) {
	if err := s.clientCacheInvalidator.Invalidate(chatID); err != nil {
		s.logger.Error(fmt.Sprintf("delete link: invalidate cache for chatID %d: %v", chatID, err))
	}
	return s.linkRepo.DeleteLink(chatID, linkURL)
}
