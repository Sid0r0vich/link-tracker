package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/api/scrapper/rest"
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
	go sched.Start()

	opts := rest.StdHTTPServerOptions{}
	handler := rest.HandlerWithOptions(api, opts)

	err := http.ListenAndServe(cfg.ServerAddr, middleware.LoggingMiddleware(handler, logger))
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
