package infrastructure

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/api/scrapper/rest"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type Updater struct {
	client  http.Client
	baseURL string
}

func NewUpdater(serverAddr string) *Updater {
	return &Updater{
		client:  http.Client{},
		baseURL: fmt.Sprintf("http://%s", serverAddr),
	}
}

func (s *Updater) SendUpdate(data *domain.UpdateResponse) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("data marshal: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/updates", s.baseURL), bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("making request to bot: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request to bot: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var respStruct rest.ApiErrorResponse
		err := json.Unmarshal(body, &respStruct)
		if err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}

		return errors.New(*respStruct.ExceptionMessage)
	}

	return nil
}
