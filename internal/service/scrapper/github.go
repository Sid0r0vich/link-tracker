package scrapper

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
)

type GithubScrapper struct {
	Token  string
	Client http.Client
	Logger *slog.Logger
}

type githubRepository struct {
	author string
	name   string
}

func NewGithubScrapper(token string, logger *slog.Logger) *GithubScrapper {
	return &GithubScrapper{
		Token:  token,
		Logger: logger,
		Client: http.Client{Timeout: 5 * time.Second},
	}
}

func getError(statusCode int) error {
	switch statusCode {
	case http.StatusTooManyRequests:
		return uerrors.ErrTooManyRequests
	case http.StatusUnauthorized:
		return uerrors.ErrBadToken
	case http.StatusForbidden:
		return uerrors.ErrInternal
	default:
		return fmt.Errorf("GitHub API error, status: %d, code: %w", statusCode, uerrors.ErrBadURL)
	}
}

func (s *GithubScrapper) makeRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, uerrors.ErrBadURL
	}

	if s.Token != "" {
		req.Header.Set("Authorization", "token "+s.Token)
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

		return nil, getError(resp.StatusCode)
	}

	return resp, nil
}

func (s *GithubScrapper) GetUpdate(url string) (*domain.Update, error) {
	repo, err := s.getRepository(url)
	if err != nil {
		return nil, fmt.Errorf("get repository: %v, %w", err, uerrors.ErrBadURL)
	}

	repoUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s", repo.author, repo.name)
	resp, err := s.makeRequest(repoUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var upd struct {
		UpdatedAt time.Time `json:"updated_at"`
		Desc      string    `json:"description"`
	}

	err = json.NewDecoder(resp.Body).Decode(&upd)
	if err != nil {
		return nil, fmt.Errorf("json decoder: %w", err)
	}

	update := domain.Update{
		URL:       url,
		UpdatedAt: upd.UpdatedAt,
	}

	return &update, nil
}

func (s *GithubScrapper) getRepository(lurl string) (*githubRepository, error) {
	u, err := url.Parse(lurl)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	if u.Scheme != "https" {
		return nil, fmt.Errorf("invalid scheme: %s", u.Scheme)
	}
	if u.Host != "github.com" {
		return nil, fmt.Errorf("invalid host: %s", u.Host)
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid path: %s", u.Path)
	}

	return &githubRepository{
		author: parts[0],
		name:   parts[1],
	}, nil
}
