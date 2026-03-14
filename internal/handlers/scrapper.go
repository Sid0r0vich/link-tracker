package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/api/scrapper/rest"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/api/scrapper/rest"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
	scrapper_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/uerrors"
)

const (
	BadRequestParams            = "Некорректные параметры запроса"
	ChatAlreadyExists           = "Чат уже существует"
	ChatNotExists               = "Чат не существует"
	LinkAlreadyExists           = "Ссылка уже отслеживается"
	InternalServerError         = "Внутренняя ошибка"
	ChatNotExistsOrLinkNotFound = "Чат не существует или ссылка не найдена"
)

func writeJSONError(w http.ResponseWriter, code int, description string, msg string) {
	writeJSONErrorWithCode(w, code, "", description, msg)
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

type UpdatesAPI struct {
	linkRepo repository.LinkRepository
	Logger   *slog.Logger
	Scrapper scrapper_repository.Scrapper
}

func NewUpdatesAPI(
	linkRepo repository.LinkRepository,
	scrapper scrapper_repository.Scrapper,
	logger *slog.Logger,
) *UpdatesAPI {
	return &UpdatesAPI{
		linkRepo: linkRepo,
		Logger:   logger,
		Scrapper: scrapper,
	}
}

func (api *UpdatesAPI) DeleteLinks(w http.ResponseWriter, r *http.Request, params api.DeleteLinksParams) {
	type Request struct {
		Link string `json:"link"`
	}

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, BadRequestParams, err.Error())
		return
	}

	link, err := api.linkRepo.DeleteLink(params.TgChatId, req.Link)
	if err != nil {
		if errors.Is(err, uerrors.ErrChatNotExists) || errors.Is(err, uerrors.ErrLinkNotFound) {
			writeJSONError(w, http.StatusNotFound, ChatNotExistsOrLinkNotFound, err.Error())
			return
		}

		writeJSONError(w, http.StatusInternalServerError, InternalServerError, err.Error())
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

func (api *UpdatesAPI) GetLinks(w http.ResponseWriter, r *http.Request, params api.GetLinksParams) {
	links, err := api.linkRepo.GetLinks(params.TgChatId)
	if err != nil {
		if errors.Is(err, uerrors.ErrChatNotExists) {
			writeJSONError(w, http.StatusNotFound, ChatNotExists, err.Error())
			return
		}

		writeJSONError(w, http.StatusInternalServerError, InternalServerError, err.Error())
		return
	}

	var n int32 = int32(len(links))

	linksResp := make([]rest.LinkResponse, n)
	for ind, link := range links {
		linksResp[ind] = rest.LinkResponse{
			Id:      &link.ID,
			Url:     &link.URL,
			Tags:    &link.Tags,
			Filters: &link.Filters,
		}
	}

	resp := &rest.ListLinksResponse{
		Links: &linksResp,
		Size:  &n,
	}

	w.Header().Set("Content-Type", "json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (api *UpdatesAPI) PostLinks(w http.ResponseWriter, r *http.Request, params api.PostLinksParams) {
	var link domain.Link
	err := json.NewDecoder(r.Body).Decode(&link)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, BadRequestParams, err.Error())
		return
	}

	update, err := api.Scrapper.GetUpdate(link.URL)
	if err != nil {
		fmt.Printf("get updates from scrapper: %v\n", err)
		writeJSONErrorWithCode(w, http.StatusBadRequest, "bad_url", BadRequestParams, "incorrect link")
		return
	}

	link.UpdatedAt = update.UpdatedAt

	id, err := api.linkRepo.AddLink(params.TgChatId, link)
	if err != nil {
		if errors.Is(err, uerrors.ErrChatNotExists) {
			writeJSONError(w, http.StatusNotFound, ChatNotExists, err.Error())
		} else if errors.Is(err, uerrors.ErrLinkAlreadyExists) {
			writeJSONError(w, http.StatusConflict, LinkAlreadyExists, err.Error())
		} else {
			writeJSONError(w, http.StatusInternalServerError, InternalServerError, err.Error())
		}

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

func (api *UpdatesAPI) DeleteTgChatId(w http.ResponseWriter, r *http.Request, id int64) {
	err := api.linkRepo.DeleteChat(id)
	if err != nil {
		if errors.Is(err, uerrors.ErrChatNotExists) {
			writeJSONError(w, http.StatusNotFound, ChatNotExists, err.Error())
			return
		}

		writeJSONError(w, http.StatusInternalServerError, InternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (api *UpdatesAPI) PostTgChatId(w http.ResponseWriter, r *http.Request, id int64) {
	err := api.linkRepo.AddChat(id)
	if err != nil {
		if errors.Is(err, uerrors.ErrChatAlreadyExists) {
			writeJSONError(w, http.StatusConflict, ChatAlreadyExists, err.Error())
			return
		}

		writeJSONError(w, http.StatusInternalServerError, InternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}
