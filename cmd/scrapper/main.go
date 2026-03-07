package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure"
)

type ErrorResponse struct {
	Desc             string   `json:"description"`
	Code             string   `json:"code"`
	ExceptionName    string   `json:"exceptionName"`
	ExceptionMessage string   `json:"exceptionMessage"`
	Stacktrace       []string `json:"stacktrace"`
}

func writeJSONError(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	resp := ErrorResponse{Code: strconv.Itoa(code)}
	json.NewEncoder(w).Encode(resp)
}

type UpdatesHandler struct {
	chatRepo infrastructure.ChatRepository
}

func (api *UpdatesHandler) AddChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest)
		return
	}

	err = api.chatRepo.AddChat(id)
	if err != nil {
		writeJSONError(w, http.StatusConflict)
		return
	}

}

func run() error {
	api := UpdatesHandler{}

	r := mux.NewRouter()
	r.HandleFunc("/tg-chat/{id:[0-9]+}", api.AddChat).Methods("POST")

	serverAddr := ""
	err := http.ListenAndServe(serverAddr, r)
	if err != nil {
		return fmt.Errorf("Server error: %w", err)
	}

	return nil
}
