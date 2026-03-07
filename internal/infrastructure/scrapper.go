package infrastructure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type Scrapper struct {
	client  http.Client
	baseURL string
}

func NewScrapper(addr string) *Scrapper {
	return &Scrapper{client: http.Client{}, baseURL: fmt.Sprintf("http://%s", addr)}
}

func (s *Scrapper) AddChat(chatID int64) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/tg-chat/%d", s.baseURL, chatID), nil)
	if err != nil {
		return fmt.Errorf("making request to scrapper: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request to scrapper: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var respStruct domain.ErrorResponse
		err := json.Unmarshal(body, &respStruct)
		if err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

func (s *Scrapper) AddLink(chatID int64, link domain.Link) error {
	type Request struct {
		Link    string   `json:"link"`
		Tags    []string `json:"tags"`
		Filters []string `json:"filters"`
	}

	data := Request{
		Link:    link.URL,
		Tags:    link.Tags,
		Filters: link.Filters,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("data marshal: %w", err)
	}

	req, err := http.NewRequest("POST", s.baseURL+"/links", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("making request to scrapper: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request to scrapper: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		var respStruct domain.AddLinkResponse
		err := json.Unmarshal(body, &respStruct)
		if err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	} else {
		var respStruct domain.ErrorResponse
		err := json.Unmarshal(body, &respStruct)
		if err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}
