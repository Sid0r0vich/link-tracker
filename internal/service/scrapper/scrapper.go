package scrapper

import (
	"log/slog"
	"net/url"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
)

//go:generate go run go.uber.org/mock/mockgen -source=scrapper.go -destination=mocks/mock.gen.go -package=mocks

type Scrapper interface {
	GetUpdate(string) (*domain.Update, error)
}

type ScrapperService struct {
	scrappers map[string]Scrapper
	updateCnt int64
}

func NewScrapperService(scrappers map[string]Scrapper) *ScrapperService {
	return &ScrapperService{
		scrappers: scrappers,
		updateCnt: 0,
	}
}

func NewScrapper(cfg *config.Config, logger *slog.Logger) Scrapper {
	return NewScrapperService(map[string]Scrapper{
		"github.com":        NewGithubScrapper(cfg.Scrapper.GithubToken, logger),
		"stackoverflow.com": NewStackoverflowScrapper(cfg.Scrapper.StackoverflowKey),
	})
}

func (s *ScrapperService) getScrapperForURL(lurl string) (Scrapper, error) {
	parsedURL, err := url.Parse(lurl)
	if err != nil {
		return nil, uerrors.ErrBadURL
	}
	dom := parsedURL.Hostname()

	scrapper, ok := s.scrappers[dom]
	if !ok {
		return nil, uerrors.ErrAPINotAlowed
	}

	return scrapper, nil
}

func (s *ScrapperService) GetUpdate(lurl string) (*domain.Update, error) {
	scrapper, err := s.getScrapperForURL(lurl)
	if err != nil {
		return nil, err
	}

	upd, err := scrapper.GetUpdate(lurl)
	if err != nil {
		return nil, err
	}

	upd.ID = s.updateCnt
	s.updateCnt++
	return upd, nil
}
