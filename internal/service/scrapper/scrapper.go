package scrapper

import (
	"net/url"

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

func (m *ScrapperService) AddScrapper(domain string, scrapper Scrapper) {
	m.scrappers[domain] = scrapper
}

func (s *ScrapperService) GetUpdate(lurl string) (*domain.Update, error) {
	parsedURL, err := url.Parse(lurl)
	if err != nil {
		return nil, uerrors.ErrBadURL
	}
	dom := parsedURL.Hostname()

	scrapper, ok := s.scrappers[dom]
	if !ok {
		return nil, uerrors.ErrAPINotAlowed
	}

	upd, err := scrapper.GetUpdate(lurl)
	if err != nil {
		return nil, err
	}

	upd.ID = s.updateCnt
	s.updateCnt++
	return upd, nil
}
