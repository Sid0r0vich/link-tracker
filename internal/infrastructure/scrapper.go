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
	client http.Client
}

func NewScrapper() *Scrapper {
	return &Scrapper{client: http.Client{}}
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

	req, err := http.NewRequest("POST", "localhost/links", bytes.NewBuffer(jsonData))
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

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	return nil
}
