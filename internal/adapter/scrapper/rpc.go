package scrapper

import (
	"context"
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rest"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type ScrapperAdapterRPC struct {
	conn   *grpc.ClientConn
	client rpc.ScrapperAPIClient
}

func NewScrapperAdapterRPC(target string, opts ...grpc.DialOption) (*ScrapperAdapterRPC, error) {
	allOpts := append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, opts...)
	conn, err := grpc.NewClient(target, allOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target %s: %v", target, err)
	}
	return &ScrapperAdapterRPC{conn: conn, client: rpc.NewScrapperAPIClient(conn)}, nil
}

func (s *ScrapperAdapterRPC) ConnClose() error {
	return s.conn.Close()
}

func (s *ScrapperAdapterRPC) AddChat(chatID int64) error {
	ctx := context.Background()

	_, err := s.client.RegisterChat(ctx, &rpc.RegisterChatRequest{Id: chatID})
	if err != nil {
		return fmt.Errorf("response: %w", err)
	}

	return nil
}

func (s *ScrapperAdapterRPC) DeleteChat(chatID int64) error {
	return nil
}

func (s *ScrapperAdapterRPC) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	ctx := context.Background()
	req := rpc.GetLinksRequest{
		TgChatId: chatID,
	}

	resp, err := s.client.GetLinks(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("response: %w", err)
	}

	links := make([]domain.LinkWithID, len(resp.GetLinks()))
	for i, link := range resp.GetLinks() {
		links[i] = *domain.LinkResponseToLinkWithID(&rest.LinkResponse{
			Id:      &link.Id,
			Url:     &link.Url,
			Tags:    &link.Tags,
			Filters: &link.Filters,
		})
	}

	return links, nil
}

func (s *ScrapperAdapterRPC) AddLink(chatID int64, link domain.Link) error {
	ctx := context.Background()
	req := rpc.AddLinkRequest{
		TgChatId: chatID,
		Filters:  link.Filters,
		Url:      link.URL,
		Tags:     link.Tags,
	}

	_, err := s.client.AddLink(ctx, &req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.AlreadyExists:
				return uerrors.ErrLinkAlreadyExists

			case codes.NotFound:
				return uerrors.ErrLinkNotFound

			case codes.InvalidArgument:
				return uerrors.ErrBadURL

			case codes.Unavailable:
				return uerrors.ErrAPIUnavailable

			case codes.PermissionDenied:
				return uerrors.ErrAPINotAlowed
			}
		}

		return uerrors.ErrInternal
	}

	return nil
}

func (s *ScrapperAdapterRPC) DeleteLink(chatID int64, url string) error {
	ctx := context.Background()
	req := rpc.RemoveLinkRequest{
		TgChatId: chatID,
		Link:     url,
	}

	_, err := s.client.RemoveLink(ctx, &req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				return uerrors.ErrChatNotExistsOrLinkNotFound
			}
		}

		return uerrors.ErrInternal
	}

	return nil
}
