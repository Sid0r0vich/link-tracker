package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers"
	link_service "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rest"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rest"
)

type ErrBadRequest struct {
	err error
}

func (errDecode ErrBadRequest) Error() string {
	return fmt.Sprintf("decode json: %v", errDecode.err)
}

func writeJSONError(w http.ResponseWriter, err error) {
	var code int
	var description string
	errBadReq := ErrBadRequest{}

	switch {
	case errors.Is(err, uerrors.ErrLinkNotFound):
		code = http.StatusNotFound
		description = handlers.LinkNotFound
	case errors.Is(err, uerrors.ErrChatNotExists):
		code = http.StatusNotFound
		description = handlers.ChatNotExists
	case errors.As(err, &errBadReq):
		code = http.StatusBadRequest
		description = handlers.BadRequestParams
	case errors.Is(err, uerrors.ErrLinkAlreadyExists):
		code = http.StatusConflict
		description = handlers.LinkAlreadyExists
	case errors.Is(err, uerrors.ErrChatAlreadyExists):
		code = http.StatusConflict
		description = handlers.ChatAlreadyExists
	case errors.Is(err, uerrors.ErrBadURL):
		writeJSONErrorWithCode(w, http.StatusBadRequest, "bad_url", handlers.BadRequestParams, "incorrect link")
		return
	case errors.Is(err, uerrors.ErrAPIUnavailable):
		writeJSONErrorWithCode(w, http.StatusInternalServerError, "api_unavailable", handlers.APIUnavailable, "api unavailable")
		return
	case errors.Is(err, uerrors.ErrAPINotAlowed):
		fmt.Fprintf(os.Stderr, "write json: api not allowed\n")
		writeJSONErrorWithCode(w, http.StatusBadRequest, "api_not_allowed", handlers.ApiNotAllowed, "api not allowed")
		return
	default:
		code = http.StatusInternalServerError
		description = handlers.InternalServerError
	}

	writeJSONErrorWithCode(w, code, "", description, err.Error())
}

func writeJSONErrorWithCode(w http.ResponseWriter, code int, error_code string, description string, msg string) {
	w.WriteHeader(code)

	resp := rest.ApiErrorResponse{
		Description:      &description,
		Code:             &error_code,
		ExceptionMessage: &msg,
	}
	json.NewEncoder(w).Encode(resp)
}

type ScrapperRestServer struct {
	Logger      *slog.Logger
	LinkService link_service.LinkService
}

func NewUpdatesRestServer(
	linkService link_service.LinkService,
	logger *slog.Logger,
) *ScrapperRestServer {
	return &ScrapperRestServer{
		Logger:      logger,
		LinkService: linkService,
	}
}

func (s *ScrapperRestServer) DeleteLinks(w http.ResponseWriter, r *http.Request, params api.DeleteLinksParams) {
	type Request struct {
		Link string `json:"link"`
	}

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSONError(w, &ErrBadRequest{err: err})
		return
	}

	link, err := s.LinkService.DeleteLink(params.TgChatId, req.Link)
	if err != nil {
		s.Logger.Error(err.Error())
		writeJSONError(w, err)
		return
	}

	resp := rest.LinkResponse{
		Id:      &link.ID,
		Url:     &link.URL,
		Tags:    &link.Tags,
		Filters: &link.Filters,
	}

	w.Header().Set("Content-Type", "json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (s *ScrapperRestServer) GetLinks(w http.ResponseWriter, r *http.Request, params api.GetLinksParams) {
	links, err := s.LinkService.GetLinks(params.TgChatId)
	if err != nil {
		s.Logger.Error(err.Error())
		writeJSONError(w, err)
		return
	}

	var n int32 = int32(len(links))

	linksResp := make([]rest.LinkResponse, n)
	for ind, link := range links {
		linksResp[ind] = *domain.LinkWithIDToLinkResponse(&link)
	}

	resp := &rest.ListLinksResponse{
		Links: &linksResp,
		Size:  &n,
	}

	w.Header().Set("Content-Type", "json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (s *ScrapperRestServer) PostLinks(w http.ResponseWriter, r *http.Request, params api.PostLinksParams) {
	var link domain.Link
	err := json.NewDecoder(r.Body).Decode(&link)
	if err != nil {
		writeJSONError(w, err)
		return
	}

	id, err := s.LinkService.AddLink(params.TgChatId, link)
	if err != nil {
		s.Logger.Error(err.Error())
		writeJSONError(w, err)
		return
	}

	resp := &rest.LinkResponse{
		Id:      &id,
		Url:     &link.URL,
		Tags:    &link.Tags,
		Filters: &link.Filters,
	}

	w.Header().Set("Content-Type", "json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (s *ScrapperRestServer) DeleteTgChatId(w http.ResponseWriter, r *http.Request, id int64) {
	err := s.LinkService.DeleteChat(id)
	if err != nil {
		writeJSONError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *ScrapperRestServer) PostTgChatId(w http.ResponseWriter, r *http.Request, id int64) {
	err := s.LinkService.AddChat(id)
	if err != nil {
		s.Logger.Error(err.Error())
		writeJSONError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
