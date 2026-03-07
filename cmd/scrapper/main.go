package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	"go.uber.org/fx"
)

func run(api *handlers.UpdatesAPI) error {
	r := mux.NewRouter()
	r.HandleFunc("/tg-chat/{id:[0-9]+}", api.AddChat).Methods("POST")
	r.HandleFunc("/tg-chat/{id:[0-9]+}", api.DeleteChat).Methods("DELETE")
	r.HandleFunc("/links", api.GetLinks).Methods("GET")
	r.HandleFunc("/links", api.AddLink).Methods("POST")
	r.HandleFunc("/links", api.DeleteLink).Methods("DELETE")

	serverAddr := ""
	err := http.ListenAndServe(serverAddr, r)
	if err != nil {
		return fmt.Errorf("Server error: %w", err)
	}

	return nil
}

func main() {
	fx.New(
		fx.NopLogger,
		fx.Provide(
			logs.NewLogger,
			handlers.NewUpdatesAPI,
		),
		fx.Invoke(run),
	).Run()
}
