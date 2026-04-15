package rpc

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers"
	link_service "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func writeRPCError(err error) error {
	var code codes.Code
	var description string

	switch {
	case errors.Is(err, uerrors.ErrLinkNotFound):
		code = codes.NotFound
		description = handlers.LinkNotFound
	case errors.Is(err, uerrors.ErrChatNotExists):
		code = codes.NotFound
		description = handlers.ChatNotExists
	case errors.Is(err, uerrors.ErrLinkAlreadyExists):
		code = codes.AlreadyExists
		description = handlers.LinkAlreadyExists
	case errors.Is(err, uerrors.ErrChatAlreadyExists):
		code = codes.AlreadyExists
		description = handlers.ChatAlreadyExists
	case errors.Is(err, uerrors.ErrBadURL):
		code = codes.InvalidArgument
		description = handlers.BadRequestParams
	case errors.Is(err, uerrors.ErrAPIUnavailable):
		code = codes.Unavailable
		description = handlers.APIUnavailable
	case errors.Is(err, uerrors.ErrAPINotAlowed):
		code = codes.PermissionDenied
		description = handlers.ApiNotAllowed
	default:
		code = http.StatusInternalServerError
		description = handlers.InternalServerError
	}

	return status.Error(code, description)
}

type ScrapperRPCServer struct {
	rpc.UnimplementedScrapperAPIServer
	LinkService link_service.LinkService
	Logger      *slog.Logger
}

func NewUpdatesRPCServer(
	linkService link_service.LinkService,
	logger *slog.Logger,
) *ScrapperRPCServer {
	return &ScrapperRPCServer{
		LinkService: linkService,
		Logger:      logger,
	}
}

func (s *ScrapperRPCServer) GetLinks(ctx context.Context, req *rpc.GetLinksRequest) (*rpc.ListLinksResponse, error) {
	links, err := s.LinkService.GetLinks(req.GetTgChatId())
	if err != nil {
		s.Logger.Error(err.Error())
		return nil, writeRPCError(err)
	}

	var n int32 = int32(len(links))

	linksResp := make([]*rpc.LinkResponse, n)
	for ind, link := range links {
		linksResp[ind] = domain.LinkWithIDToRPCLinkResponse(&link)
	}

	resp := &rpc.ListLinksResponse{
		Links: linksResp,
		Size:  int32(n),
	}

	return resp, nil
}

func (s *ScrapperRPCServer) RegisterChat(ctx context.Context, req *rpc.RegisterChatRequest) (*emptypb.Empty, error) {
	err := s.LinkService.AddChat(req.GetId())
	if err != nil {
		s.Logger.Error(err.Error())
		return nil, writeRPCError(err)
	}

	return &emptypb.Empty{}, nil
}
func (s *ScrapperRPCServer) DeleteChat(ctx context.Context, req *rpc.DeleteChatRequest) (*emptypb.Empty, error) {
	err := s.LinkService.DeleteChat(req.GetId())
	if err != nil {
		s.Logger.Error(err.Error())
		return nil, writeRPCError(err)
	}

	return &emptypb.Empty{}, nil
}
func (s *ScrapperRPCServer) AddLink(ctx context.Context, req *rpc.AddLinkRequest) (*rpc.LinkResponse, error) {
	link := domain.RPCLinkResponseToLink(req)

	id, err := s.LinkService.AddLink(req.GetTgChatId(), *link)
	if err != nil {
		s.Logger.Error(err.Error())
		return nil, writeRPCError(err)
	}

	resp := &rpc.LinkResponse{
		Id:      id,
		Url:     link.URL,
		Tags:    link.Tags,
		Filters: link.Filters,
	}

	return resp, nil
}
func (s *ScrapperRPCServer) RemoveLink(ctx context.Context, req *rpc.RemoveLinkRequest) (*rpc.LinkResponse, error) {
	link, err := s.LinkService.DeleteLink(req.TgChatId, req.Link)
	if err != nil {
		s.Logger.Error(err.Error())
		return nil, writeRPCError(err)
	}

	resp := rpc.LinkResponse{
		Id:      link.ID,
		Url:     link.URL,
		Tags:    link.Tags,
		Filters: link.Filters,
	}

	return &resp, nil
}
