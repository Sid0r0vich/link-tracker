package infrastructure

import (
	"bytes"
	"encoding/json"
	"errors"
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

func (s *Scrapper) makeRequest(method string, url string, reqBody io.Reader, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("making request to scrapper: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to scrapper: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var respStruct domain.ErrorResponse
		err := json.Unmarshal(body, &respStruct)
		if err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		return nil, errors.New(respStruct.ExceptionMessage)
	}

	return body, nil
}

func NewScrapper(addr string) *Scrapper {
	return &Scrapper{client: http.Client{}, baseURL: fmt.Sprintf("http://%s", addr)}
}

func (s *Scrapper) AddChat(chatID int64) error {
	_, err := s.makeRequest("POST", fmt.Sprintf("%s/tg-chat/%d", s.baseURL, chatID), nil, nil)
	return err
}

func (s *Scrapper) DeleteChat(chatID int64) error {
	_, err := s.makeRequest("DELETE", fmt.Sprintf("%s/tg-chat/%d", s.baseURL, chatID), nil, nil)
	return err
}

func (s *Scrapper) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	body, err := s.makeRequest(
		"GET",
		fmt.Sprintf("%s/links", s.baseURL),
		nil,
		map[string]string{"Tg-Chat-Id": strconv.FormatInt(chatID, 10)},
	)
	if err != nil {
		return nil, err
	}

	var respStruct domain.LinksResponse
	err = json.Unmarshal(body, &respStruct)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	res := make([]domain.LinkWithID, len(respStruct.Links))
	for ind, link := range respStruct.Links {
		res[ind] = domain.LinkWithID{
			Link: domain.Link{
				LinkInfo: domain.LinkInfo{
					Tags:    link.Tags,
					Filters: link.Filters,
				},
				URL: link.URL,
			},
			ID: link.ID,
		}
	}

	return res, nil
}

func (s *Scrapper) AddLink(chatID int64, link domain.Link) (int64, error) {
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
		return 0, fmt.Errorf("data marshal: %w", err)
	}

	body, err := s.makeRequest(
		"POST",
		fmt.Sprintf("%s/links", s.baseURL),
		bytes.NewBuffer(jsonData),
		map[string]string{"Content-Type": "application/json", "Tg-Chat-Id": strconv.FormatInt(chatID, 10)},
	)
	if err != nil {
		return 0, err
	}

	var respStruct domain.LinkResponse
	err = json.Unmarshal(body, &respStruct)
	if err != nil {
		return 0, fmt.Errorf("unmarshal response: %w", err)
	}

	return respStruct.ID, nil
}

func (s *Scrapper) DeleteLink(chatID int64, url string) (*domain.LinkWithID, error) {
	type Request struct {
		Link string `json:"link"`
	}

	data := Request{Link: url}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("data marshal: %w", err)
	}

	body, err := s.makeRequest(
		"DELETE",
		fmt.Sprintf("%s/links", s.baseURL),
		bytes.NewBuffer(jsonData),
		map[string]string{"Content-Type": "application/json", "Tg-Chat-Id": strconv.FormatInt(chatID, 10)},
	)
	if err != nil {
		return nil, err
	}

	var respStruct domain.LinkResponse
	err = json.Unmarshal(body, &respStruct)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &domain.LinkWithID{
		Link: domain.Link{
			LinkInfo: domain.LinkInfo{
				Tags:    respStruct.Tags,
				Filters: respStruct.Filters,
			},
			URL: respStruct.URL,
		},
		ID: respStruct.ID,
	}, nil
}
