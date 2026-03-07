package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

const (
	BadRequestParams    = "Некорректные параметры запроса"
	ChatAlreadyExists   = "Чат уже существует"
	ChatNotExists       = "Чат не существует"
	LinkAlreadyExists   = "Ссылка уже отслеживается"
	InternalServerError = "Внутренняя ошибка"
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
	linkRepo LinkRepository
	logger   *slog.Logger
}

func NewUpdatesAPI(logger *slog.Logger) *UpdatesAPI {
	return &UpdatesAPI{
		logger: logger,
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
		if errors.Is(err, ErrChatAlreadyExists) {
			writeJSONError(w, http.StatusConflict, ChatAlreadyExists, err.Error())
			return
		}

		writeJSONError(w, http.StatusInternalServerError, InternalServerError, err.Error())
	}
}

func (api *UpdatesAPI) DeleteChat(w http.ResponseWriter, r *http.Request) {

}

func (api *UpdatesAPI) GetLinks(w http.ResponseWriter, r *http.Request) {

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

	err = api.linkRepo.AddLink(chatID, link)
	if err != nil {
		if errors.Is(err, ErrChatNotExists) {
			writeJSONError(w, http.StatusNotFound, ChatNotExists, err.Error())
		} else if errors.Is(err, ErrLinkAlreadyExists) {
			writeJSONError(w, http.StatusConflict, LinkAlreadyExists, err.Error())
		} else {
			writeJSONError(w, http.StatusInternalServerError, InternalServerError, err.Error())
		}

		return
	}
}

func (api *UpdatesAPI) DeleteLink(w http.ResponseWriter, r *http.Request) {

}
