package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository"
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
	w.WriteHeader(code)

	resp := domain.ErrorResponse{
		Description:      description,
		Code:             strconv.Itoa(code),
		ExceptionMessage: msg,
	}
	json.NewEncoder(w).Encode(resp)
}

type UpdatesAPI struct {
	linkRepo repository.LinkRepository
	logger   *slog.Logger
}

func NewUpdatesAPI(linkRepo repository.LinkRepository, logger *slog.Logger) *UpdatesAPI {
	return &UpdatesAPI{
		linkRepo: linkRepo,
		logger:   logger,
	}
}

func (api *UpdatesAPI) AddChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, BadRequestParams, err.Error())
		return
	}

	err = api.linkRepo.AddChat(id)
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

func (api *UpdatesAPI) DeleteChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, BadRequestParams, err.Error())
		return
	}

	err = api.linkRepo.DeleteChat(id)
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

func (api *UpdatesAPI) GetLinks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	chatIDStr := r.Header.Get("Tg-Chat-Id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, BadRequestParams, err.Error())
		return
	}

	links, err := api.linkRepo.GetLinks(chatID)
	if err != nil {
		if errors.Is(err, uerrors.ErrChatNotExists) {
			writeJSONError(w, http.StatusNotFound, ChatNotExists, err.Error())
			return
		}

		writeJSONError(w, http.StatusInternalServerError, InternalServerError, err.Error())
		return
	}

	linksResp := make([]domain.LinkResponse, len(links))
	for ind, link := range links {
		linksResp[ind] = domain.LinkResponse{
			ID:      link.ID,
			URL:     link.URL,
			Tags:    link.Tags,
			Filters: link.Filters,
		}
	}

	resp := domain.LinksResponse{
		Links: linksResp,
		Size:  len(linksResp),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (api *UpdatesAPI) AddLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	chatIDStr := r.Header.Get("Tg-Chat-Id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, BadRequestParams, err.Error())
		return
	}

	var link domain.Link
	err = json.NewDecoder(r.Body).Decode(&link)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, BadRequestParams, err.Error())
		return
	}

	id, err := api.linkRepo.AddLink(chatID, link)
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

	resp := domain.LinkResponse{
		ID:      id,
		URL:     link.URL,
		Tags:    link.Tags,
		Filters: link.Filters,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (api *UpdatesAPI) DeleteLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	chatIDStr := r.Header.Get("Tg-Chat-Id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, BadRequestParams, err.Error())
		return
	}

	type Request struct {
		Link string `json:"link"`
	}

	var req Request
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, BadRequestParams, err.Error())
		return
	}

	link, err := api.linkRepo.DeleteLink(chatID, req.Link)
	if err != nil {
		if errors.Is(err, uerrors.ErrChatNotExists) || errors.Is(err, uerrors.ErrLinkNotFound) {
			writeJSONError(w, http.StatusBadRequest, ChatNotExistsOrLinkNotFound, err.Error())
			return
		}

		writeJSONError(w, http.StatusInternalServerError, InternalServerError, err.Error())
		return
	}

	resp := domain.LinkResponse{
		ID:      link.ID,
		URL:     link.URL,
		Tags:    link.Tags,
		Filters: link.Filters,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
