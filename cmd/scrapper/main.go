package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/middleware"
	link_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
	scrapper_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scheduler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper"
	"go.uber.org/fx"
)

type Config struct {
	ServerAddr    string
	GithubToken   string
	BotServerAddr string
}

func loadConfig(logger *slog.Logger) (*Config, error) {
	logger.Info("load config")

	serverAddr := os.Getenv("SERVER_ADDR")
	githubToken := os.Getenv("GITHUB_TOKEN")
	botServerAddr := os.Getenv("BOT_SERVER_ADDR")
	if serverAddr == "" || githubToken == "" || botServerAddr == "" {
		logger.Error("fail to load config", "error", "empty value")
		return nil, fmt.Errorf("environment error")
	}

	return &Config{ServerAddr: serverAddr, GithubToken: githubToken, BotServerAddr: botServerAddr}, nil
}

func run(cfg *Config, api *handlers.UpdatesAPI, logger *slog.Logger, sched *scheduler.Scheduler) error {
	r := mux.NewRouter()
	r.HandleFunc("/tg-chat/{id:[0-9]+}", api.AddChat).Methods("POST")
	r.HandleFunc("/tg-chat/{id:[0-9]+}", api.DeleteChat).Methods("DELETE")
	r.HandleFunc("/links", api.GetLinks).Methods("GET")
	r.HandleFunc("/links", api.AddLink).Methods("POST")
	r.HandleFunc("/links", api.DeleteLink).Methods("DELETE")

	go sched.Start()

	err := http.ListenAndServe(cfg.ServerAddr, middleware.LoggingMiddleware(r, logger))
	if err != nil {
		logger.Error("fail to start server", "error", err)
	}

	return nil
}

func main() {
	fx.New(
		//fx.NopLogger,
		fx.Provide(
			loadConfig,
			logs.NewLogger,
			link_repository.NewInMemoryLinkRepo,
			func(repo *link_repository.InMemoryLinkRepo) link_repository.LinkRepository {
				return repo
			},
			func(repo *link_repository.InMemoryLinkRepo) link_repository.LinkUpdateRepository {
				return repo
			},
			func(cfg *Config) *infrastructure.Updater {
				return infrastructure.NewUpdater(cfg.BotServerAddr)
			},
			handlers.NewUpdatesAPI,
			func(cfg *Config) scrapper_repository.Scrapper {
				return scrapper.NewGithubSrcapper(cfg.GithubToken)
			},
			scheduler.NewScheduler,
		),
		fx.Invoke(run),
	).Run()
}
