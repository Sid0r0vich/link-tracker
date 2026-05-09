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

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
)

type GithubScrapper struct {
	httpCfg   *config.HTTPConfig
	token     string
	Client    utils.Client
	logger    *slog.Logger
	ApiHost   string
	ApiScheme string
	urlHost   string
	urlScheme string
}

type githubRepository struct {
	author string
	name   string
}

func NewGithubScrapper(cfg *config.HTTPConfig, cbCfg *config.CircuitBreakerConfig, token string, logger *slog.Logger) *GithubScrapper {
	return &GithubScrapper{
		httpCfg:   cfg,
		token:     token,
		logger:    logger,
		Client:    utils.NewRetryClient(utils.NewCircuitBreakerClient(&http.Client{Timeout: cfg.Timeout}, cbCfg, logger), cfg, logger),
		ApiHost:   "api.github.com",
		ApiScheme: "https",
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

	if s.token != "" {
		req.Header.Set("Authorization", "token "+s.token)
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
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

	repoUrl := fmt.Sprintf("%s://%s/repos/%s/%s", s.ApiScheme, s.ApiHost, repo.author, repo.name)
	resp, err := s.makeRequest(repoUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var upd struct {
		Desc string `json:"description"`
	}

	err = json.NewDecoder(resp.Body).Decode(&upd)
	if err != nil {
		return nil, fmt.Errorf("json decoder: %w", err)
	}

	pulls, err := s.getEvents(repo, "pulls")
	if err != nil {
		return nil, fmt.Errorf("get pulls: %w", err)
	}

	issues, err := s.getEvents(repo, "issues")
	if err != nil {
		return nil, fmt.Errorf("get issues: %w", err)
	}

	allEvents := append(pulls, issues...)

	updatedAt := time.Time{}
	for _, pl := range allEvents {
		s.logger.Info("event request update time", "url", url, "created_at", pl.CreatedAt)
		if pl.CreatedAt.After(updatedAt) {
			updatedAt = pl.CreatedAt
		}
	}

	update := domain.Update{
		URL:       url,
		UpdatedAt: updatedAt,
		Data:      allEvents,
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

func (s *GithubScrapper) getEvents(repo *githubRepository, typ string) ([]domain.Event, error) {
	type user struct {
		Login string `json:"login"`
	}
	type pull struct {
		CreatedAt   time.Time `json:"created_at"`
		Title       string    `json:"title"`
		User        user      `json:"user"`
		Description string    `json:"body"`
	}

	var url string
	var humanType string
	switch typ {
	case "pulls":
		url = fmt.Sprintf("%s://%s/repos/%s/%s/pulls", s.ApiScheme, s.ApiHost, repo.author, repo.name)
		humanType = "pull request"
	case "issues":
		url = fmt.Sprintf("%s://%s/repos/%s/%s/issues", s.ApiScheme, s.ApiHost, repo.author, repo.name)
		humanType = "issue"
	default:
		return nil, fmt.Errorf("invalid event type: %s", typ)
	}

	resp, err := s.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pulls []pull
	err = json.NewDecoder(resp.Body).Decode(&pulls)
	if err != nil {
		return nil, fmt.Errorf("json decoder: %w", err)
	}

	result := make([]domain.Event, 0)
	for _, pl := range pulls {
		result = append(result, domain.Event{
			Type:        humanType,
			CreatedAt:   pl.CreatedAt,
			Title:       pl.Title,
			Username:    pl.User.Login,
			Description: utils.CutDescription(pl.Description, MaxDescriptionLength),
		})
	}

	return result, nil
}
