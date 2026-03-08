package scrapper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/uerrors"
)

type GithubScrapper struct {
	Token  string
	Client http.Client
}

func NewGithubSrcapper(token string) *GithubScrapper {
	return &GithubScrapper{
		Token:  token,
		Client: http.Client{},
	}
}

func (s *GithubScrapper) GetUpdate(url string) (*domain.Update, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if s.Token != "" {
		req.Header.Set("Authorization", "token "+s.Token)
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body: %w", err)
		}

		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			return nil, uerrors.ErrTooManyRequests
		case http.StatusUnauthorized:
			return nil, uerrors.ErrBadToken
		default:
			return nil, fmt.Errorf("GitHub API error, status code: %w", uerrors.ErrBadURL)
		}
	}

	var upd struct {
		UpdatedAt time.Time `json:"updated_at"`
		Desc      string    `json:"description"`
	}

	err = json.NewDecoder(resp.Body).Decode(&upd)
	if err != nil {
		return nil, fmt.Errorf("json decoder: %w", err)
	}

	update := domain.Update{
		ID:        0,
		URL:       url,
		Desc:      upd.Desc,
		UpdatedAt: upd.UpdatedAt,
	}

	return &update, nil
}
