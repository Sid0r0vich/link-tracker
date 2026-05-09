package update

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

type UpdateRestService struct {
	client api.ClientWithResponsesInterface
}

func NewUpdateRestService(serverAddr string, cfg *config.HTTPConfig) (*UpdateRestService, error) {
	c, err := api.NewClientWithResponses(serverAddr, api.WithHTTPClient(&http.Client{Timeout: cfg.Timeout}))
	if err != nil {
		return nil, fmt.Errorf("update service create: %w", err)
	}

	return &UpdateRestService{client: c}, nil
}

func (s *UpdateRestService) SendUpdate(data *domain.UpdateMessage) error {
	ctx := context.Background()

	resp, err := s.client.PostUpdatesWithResponse(ctx, api.PostUpdatesJSONRequestBody{
		Id:        data.Id,
		TgChatIds: data.TgChatIds,
		Url:       data.Url,
		Data:      domain.EventSliceToApiEventSlice(data.Data),
	})
	if err != nil {
		return fmt.Errorf("response: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return errors.New(*resp.JSON400.ExceptionMessage)
	default:
		return fmt.Errorf("unexpected update status code: %d", resp.StatusCode())
	}
}
