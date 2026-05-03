package scrapper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rest"
)

type ScrapperRestAdapter struct {
	Client rest.ClientWithResponsesInterface
}

func NewScrapperAdapterRest(baseURL string) (*ScrapperRestAdapter, error) {
	c, err := rest.NewClientWithResponses(baseURL)
	if err != nil {
		return nil, fmt.Errorf("scrapper service create: %w", err)
	}

	return &ScrapperRestAdapter{Client: c}, nil
}

func (s *ScrapperRestAdapter) AddChat(chatID int64) error {
	ctx := context.Background()

	resp, err := s.Client.PostTgChatIdWithResponse(ctx, chatID)
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

func (s *ScrapperRestAdapter) DeleteChat(chatID int64) error {
	ctx := context.Background()

	resp, err := s.Client.DeleteTgChatIdWithResponse(ctx, chatID)
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

func (s *ScrapperRestAdapter) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	ctx := context.Background()
	params := rest.GetLinksParams{TgChatId: chatID}

	resp, err := s.Client.GetLinksWithResponse(ctx, &params)
	if err != nil {
		return nil, fmt.Errorf("response: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		links := domain.LinkResponseSliceToLinkWithID(*resp.JSON200.Links)

		return links, nil

	default:
		return nil, uerrors.ErrInternal
	}
}

func (s *ScrapperRestAdapter) AddLink(chatID int64, link domain.Link) error {
	ctx := context.Background()
	params := rest.PostLinksParams{TgChatId: chatID}
	body := rest.PostLinksJSONRequestBody{
		Filters: &link.Filters,
		Link:    &link.URL,
		Tags:    &link.Tags,
	}

	resp, err := s.Client.PostLinksWithResponse(ctx, &params, body)
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
		if resp.JSON400 != nil {
			if resp.JSON400.Code == nil {
				return fmt.Errorf("unexpected error response without code")
			}

			switch *resp.JSON400.Code {
			case "bad_url":
				return uerrors.ErrBadURL

			case "api_not_allowed":
				return uerrors.ErrAPINotAlowed
			}
		}

	case http.StatusInternalServerError:
		var errResp rest.ApiErrorResponse
		err = json.Unmarshal(resp.Body, &errResp)
		if err != nil {
			return fmt.Errorf("unmarshal error response: %w", err)
		}

		if errResp.Code == nil {
			return fmt.Errorf("unexpected error response without code")
		}

		switch *errResp.Code {
		case "api_unavailable":
			return uerrors.ErrAPIUnavailable
		}
	}

	return uerrors.ErrInternal
}

func (s *ScrapperRestAdapter) DeleteLink(chatID int64, url string) error {
	ctx := context.Background()
	params := rest.DeleteLinksParams{TgChatId: chatID}
	body := rest.DeleteLinksJSONRequestBody{
		Link: &url,
	}

	resp, err := s.Client.DeleteLinksWithResponse(ctx, &params, body)
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
