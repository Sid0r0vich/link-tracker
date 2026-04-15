package update

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

type UpdateService struct {
	client api.ClientWithResponsesInterface
}

func NewUpdateService(serverAddr string) *UpdateService {
	c, err := api.NewClientWithResponses(serverAddr)
	if err != nil {
		panic(fmt.Sprintf("update service create: %v", err))
	}

	return &UpdateService{client: c}
}

func (s *UpdateService) SendUpdate(data *api.UpdateResponse) error {
	ctx := context.Background()

	resp, err := s.client.PostUpdatesWithResponse(ctx, *data)
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
