package scrapper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
)

type StackoverflowScrapper struct {
	Key    string
	Client http.Client
}

func NewStackoverflowScrapper(key string) *StackoverflowScrapper {
	return &StackoverflowScrapper{
		Key:    key,
		Client: http.Client{Timeout: 5 * time.Second},
	}
}

func (s *StackoverflowScrapper) makeRequest(url string) (*http.Response, error) {
	newURL := fmt.Sprintf("%s?key=%s&site=stackoverflow&filter=withbody", url, s.Key)
	req, err := http.NewRequest("GET", newURL, nil)
	if err != nil {
		return nil, uerrors.ErrBadURL
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		if utils.IsNetError(err) {
			return nil, uerrors.ErrAPIUnavailable
		}
		return nil, uerrors.ErrBadURL
	}

	if resp.StatusCode != http.StatusOK {
		_, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body: %w", err)
		}

		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, uerrors.ErrBadToken
		case http.StatusForbidden:
			return nil, uerrors.ErrInternal
		default:
			return nil, fmt.Errorf("Stack Overflow API error, url: %s, status: %d, code: %w", newURL, resp.StatusCode, uerrors.ErrBadURL)
		}
	}

	return resp, nil
}

func (s *StackoverflowScrapper) GetUpdate(rurl string) (*domain.Update, error) {
	parsedUrl, err := url.Parse(rurl)
	if err != nil {
		return nil, fmt.Errorf("cannot parse url: %w", uerrors.ErrBadURL)
	}
	parsedUrl.RawQuery = ""
	lurl := parsedUrl.String()

	type item struct {
		LastActivityDate int64  `json:"last_activity_date"`
		Title            string `json:"title"`
	}
	var upd struct {
		Items []item `json:"items"`
	}

	resp, err := s.makeRequest(lurl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&upd)
	if err != nil {
		return nil, fmt.Errorf("json decoder: %w", err)
	}

	if len(upd.Items) == 0 {
		return nil, fmt.Errorf("no items")
	}
	it := upd.Items[0]

	update := domain.Update{
		URL:       lurl,
		UpdatedAt: time.Unix(it.LastActivityDate, 0),
	}

	return &update, nil
}
