package scrapper

import (
	"context"
	"fmt"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/api/scrapper/rest"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/uerrors"
)

type ScrapperAdapterImpl struct {
	client rest.ClientWithResponsesInterface
}

func NewScrapperAdapterImpl(baseURL string) (*ScrapperAdapterImpl, error) {
	c, err := rest.NewClientWithResponses(baseURL)
	if err != nil {
		return nil, fmt.Errorf("scrapper service create: %w", err)
	}

	return &ScrapperAdapterImpl{client: c}, nil
}

func (s *ScrapperAdapterImpl) AddChat(chatID int64) error {
	ctx := context.Background()

	resp, err := s.client.PostTgChatIdWithResponse(ctx, chatID)
	if err != nil {
		return fmt.Errorf("response: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return nil

	default:
		return uerrors.ErrInternal
	}
}

func (s *ScrapperAdapterImpl) DeleteChat(chatID int64) error {
	ctx := context.Background()

	resp, err := s.client.DeleteTgChatIdWithResponse(ctx, chatID)
	if err != nil {
		return fmt.Errorf("response: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return nil

	default:
		return uerrors.ErrInternal
	}
}

func (s *ScrapperAdapterImpl) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	ctx := context.Background()
	params := rest.GetLinksParams{TgChatId: chatID}

	resp, err := s.client.GetLinksWithResponse(ctx, &params)
	if err != nil {
		return nil, fmt.Errorf("response: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		links := make([]domain.LinkWithID, len(*resp.JSON200.Links))
		for idx, link := range *resp.JSON200.Links {
			links[idx] = domain.LinkWithID{
				Link: domain.Link{
					LinkInfo: domain.LinkInfo{
						Tags:    *link.Tags,
						Filters: *link.Filters,
					},
					URL: *link.Url,
				},
				ID: *link.Id,
			}
		}

		return links, nil

	default:
		return nil, uerrors.ErrInternal
	}
}

func (s *ScrapperAdapterImpl) AddLink(chatID int64, link domain.Link) error {
	ctx := context.Background()
	params := rest.PostLinksParams{TgChatId: chatID}
	body := rest.PostLinksJSONRequestBody{
		Filters: &link.Filters,
		Link:    &link.URL,
		Tags:    &link.Tags,
	}

	resp, err := s.client.PostLinksWithResponse(ctx, &params, body)
	if err != nil {
		return fmt.Errorf("response: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return nil
	case http.StatusConflict:
		return uerrors.ErrLinkAlreadyExists

	case http.StatusNotFound:
		return uerrors.ErrLinkNotFound

	case http.StatusBadRequest:
		if resp.JSON400.Code != nil && *resp.JSON400.Code == "bad_url" {
			return uerrors.ErrBadURL
		}

	default:
	}

	return uerrors.ErrInternal
}

func (s *ScrapperAdapterImpl) DeleteLink(chatID int64, url string) error {
	ctx := context.Background()
	params := rest.DeleteLinksParams{TgChatId: chatID}
	body := rest.DeleteLinksJSONRequestBody{
		Link: &url,
	}

	resp, err := s.client.DeleteLinksWithResponse(ctx, &params, body)
	if err != nil {
		return fmt.Errorf("response: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return uerrors.ErrChatNotExistsOrLinkNotFound
	}

	return uerrors.ErrInternal
}
