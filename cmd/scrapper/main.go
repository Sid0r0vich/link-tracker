package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	link_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
	"go.uber.org/fx"
)

type Config struct {
	ServerAddr string
}

func loadConfig(logger *slog.Logger) (*Config, error) {
	logger.Info("load config")

	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		logger.Error("fail to load config", "error", "empty server address")
		return nil, fmt.Errorf("environment error")
	}

	return &Config{ServerAddr: serverAddr}, nil
}

func loggingMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("request", "from", r.Host, "method", r.Method, "URL", r.URL)
		next.ServeHTTP(w, r)
	})
}

func run(cfg *Config, api *handlers.UpdatesAPI, logger *slog.Logger) error {
	r := mux.NewRouter()
	r.HandleFunc("/tg-chat/{id:[0-9]+}", api.AddChat).Methods("POST")
	r.HandleFunc("/tg-chat/{id:[0-9]+}", api.DeleteChat).Methods("DELETE")
	r.HandleFunc("/links", api.GetLinks).Methods("GET")
	r.HandleFunc("/links", api.AddLink).Methods("POST")
	r.HandleFunc("/links", api.DeleteLink).Methods("DELETE")

	err := http.ListenAndServe(cfg.ServerAddr, loggingMiddleware(r, logger))
	if err != nil {
		logger.Error("fail to start server", "error", err)
	}

	return nil
}

func main() {
	fx.New(
		fx.NopLogger,
		fx.Provide(
			loadConfig,
			logs.NewLogger,
			link_repository.NewInMemoryLinkRepo,
			func(repo *link_repository.InMemoryLinkRepo, logger *slog.Logger) *handlers.UpdatesAPI {
				return handlers.NewUpdatesAPI(repo, logger)
			},
		),
		fx.Invoke(run),
	).Run()
}
