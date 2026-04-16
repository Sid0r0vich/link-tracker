package scrapper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

type StackoverflowScrapper struct {
	Key       string
	Client    http.Client
	apiHost   string
	apiScheme string
}

func NewStackoverflowScrapper(key string) *StackoverflowScrapper {
	return &StackoverflowScrapper{
		Key:       key,
		Client:    http.Client{Timeout: 5 * time.Second},
		apiHost:   "api.stackexchange.com",
		apiScheme: "https",
	}
}

func (s *StackoverflowScrapper) makeRequest(rurl string) (*http.Response, error) {
	parsedUrl, err := url.Parse(rurl)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	params := url.Values{}
	params.Add("site", "stackoverflow")
	params.Add("filter", "withbody")
	if s.Key != "" {
		params.Add("key", s.Key)
	}
	parsedUrl.RawQuery = params.Encode()
	newUrl := parsedUrl.String()

	req, err := http.NewRequest("GET", newUrl, nil)
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
			return nil, fmt.Errorf("Stack Overflow API error, url: %s, status: %d, code: %w", newUrl, resp.StatusCode, uerrors.ErrBadURL)
		}
	}

	return resp, nil
}

func (s *StackoverflowScrapper) GetUpdate(rurl string) (*domain.Update, error) {
	type item struct {
		LastActivityDate int64  `json:"last_activity_date"`
		Title            string `json:"title"`
	}
	var upd struct {
		Items []item `json:"items"`
	}

	questionID, err := s.getQuestionId(rurl)
	if err != nil {
		return nil, fmt.Errorf("get question id: %v, %w", err, uerrors.ErrBadURL)
	}

	questionUrl := fmt.Sprintf("%s://%s/questions/%s", s.apiScheme, s.apiHost, questionID)
	resp, err := s.makeRequest(questionUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&upd)
	if err != nil {
		return nil, fmt.Errorf("json decoder: %w", err)
	}

	if len(upd.Items) == 0 {
		return nil, fmt.Errorf("no items: %w", uerrors.ErrBadURL)
	}
	it := upd.Items[0]

	answers, err := s.getEvents(questionID, "answer", it.Title)
	if err != nil {
		return nil, fmt.Errorf("get events: %w", err)
	}

	comments, err := s.getEvents(questionID, "comment", it.Title)
	if err != nil {
		return nil, fmt.Errorf("get events: %w", err)
	}

	allEvents := append(answers, comments...)

	updatedAt := time.Time{}
	for _, event := range allEvents {
		if event.CreatedAt.After(updatedAt) {
			updatedAt = event.CreatedAt
		}
	}

	update := domain.Update{
		URL:       rurl,
		UpdatedAt: time.Unix(it.LastActivityDate, 0),
		Data:      allEvents,
	}

	return &update, nil
}

func (s *StackoverflowScrapper) getQuestionId(lurl string) (string, error) {
	u, err := url.Parse(lurl)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}

	if u.Scheme != "https" {
		return "", fmt.Errorf("invalid scheme: %s", u.Scheme)
	}
	if u.Host != "stackoverflow.com" {
		return "", fmt.Errorf("invalid host: %s", u.Host)
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "questions" {
		return "", fmt.Errorf("invalid path: %s", u.Path)
	}

	return parts[1], nil
}

func (s *StackoverflowScrapper) getEvents(questionID string, typ string, title string) ([]api.Event, error) {
	type Owner struct {
		DisplayName string `json:"display_name"`
	}

	type Answer struct {
		LastActivityDate int64  `json:"last_activity_date"`
		CreationDate     int64  `json:"creation_date"`
		Owner            Owner  `json:"owner"`
		Content          string `json:"body"`
	}

	var apiResponse struct {
		Items []Answer `json:"items"`
	}

	var url string
	var humanType string
	switch typ {
	case "answer":
		url = fmt.Sprintf("%s://%s/questions/%s/answers", s.apiScheme, s.apiHost, questionID)
		humanType = "answer"
	case "comment":
		url = fmt.Sprintf("%s://%s/questions/%s/comments", s.apiScheme, s.apiHost, questionID)
		humanType = "comment"
	default:
		return nil, fmt.Errorf("invalid event type: %s", typ)
	}

	resp, err := s.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	if err != nil {
		return nil, fmt.Errorf("json decoder: %w", err)
	}

	events := make([]api.Event, 0)
	for _, answer := range apiResponse.Items {
		events = append(events, api.Event{
			Type:        humanType,
			CreatedAt:   time.Unix(answer.CreationDate, 0),
			Title:       title,
			Username:    answer.Owner.DisplayName,
			Description: utils.CutDescription(answer.Content, maxDescriptionLength),
		})
	}

	return events, nil
}
