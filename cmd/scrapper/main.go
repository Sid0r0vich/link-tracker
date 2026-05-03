package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/cache"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	rest_handlers "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rest"
	rpc_handlers "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rpc"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/middleware"
	link_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scheduler"
	link_service "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/update"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rest"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rpc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

func run(
	cfg *config.Config,
	logger *slog.Logger,
	sched *scheduler.Scheduler,
	restAPI *rest_handlers.ScrapperRestServer,
	rpcAPI *rpc_handlers.ScrapperRPCServer,
) error {
	sched.Start()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("interrupt signal received. Shutdown")
		sched.Shutdown()
	}()

	if err := os.WriteFile("/tmp/ready", []byte("1"), 0644); err != nil {
		return fmt.Errorf("write ready file: %w", err)
	}

	switch cfg.Scrapper.TransportProtocol {
	case config.TransportProtocolHTTP:
		handler := rest.HandlerWithOptions(restAPI, rest.StdHTTPServerOptions{})

		err := http.ListenAndServe(cfg.Scrapper.ServerAddr, middleware.LoggingMiddleware(handler, logger))
		if err != nil {
			return fmt.Errorf("start server: %w", err)
		}

	case config.TransportProtocolGRPC:
		lis, err := net.Listen("tcp", cfg.Scrapper.ServerAddr)
		if err != nil {
			return fmt.Errorf("failed to listen: %v", err)
		}

		grpcServer := grpc.NewServer()
		rpc.RegisterScrapperAPIServer(grpcServer, rpcAPI)

		if err := grpcServer.Serve(lis); err != nil {
			return fmt.Errorf("start server: %w", err)
		}
	}

	return nil
}

func NewApp() *fx.App {
	return fx.New(
		fx.Provide(
			utils.GetContext,
			config.LoadConfig,
			logs.NewLogger,
			fx.Annotate(
				func(
					cfg *config.Config,
					lifecycle fx.Lifecycle,
					logger *slog.Logger,
				) (link_repository.LinkUnitedRepository, error) {
					var repo link_repository.LinkUnitedRepository
					var close func() error
					var err error
					switch cfg.Scrapper.DBAccessType {
					case "SQL":
						repo, close, err = link_repository.NewSQLRepo(&cfg.Database, logger)
						if err != nil {
							return nil, fmt.Errorf("failed to create SQL repo: %w", err)
						}
					case "ORM":
						repo, close, err = link_repository.NewORMRepo(&cfg.Database, logger)
						if err != nil {
							return nil, fmt.Errorf("failed to create ORM repo: %w", err)
						}
					default:
						return nil, fmt.Errorf("invalid db access type: %s", cfg.Scrapper.DBAccessType)
					}

					lifecycle.Append(fx.Hook{
						OnStop: func(context.Context) error {
							return close()
						},
					})

					return repo, nil
				},
				fx.As(new(link_repository.LinkRepository)),
				fx.As(new(link_repository.LinkUpdateRepository)),
			),
			func(
				ctx context.Context,
				cfg *config.Config,
				lifecycle fx.Lifecycle,
				logger *slog.Logger,
			) (scheduler.Updater, error) {
				switch cfg.Scrapper.UpdateCommunicationType {
				case config.UpdateCommunicationTypeHTTP:
					return update.NewUpdateRestService(fmt.Sprintf("http://%s", cfg.Bot.ServerAddr))
				case config.UpdateCommunicationTypeKafka:
					return update.NewUpdateBrokerService(ctx, &cfg.Kafka, logger)
				default:
					return nil, fmt.Errorf("invalid update communication type: %s", cfg.Scrapper.UpdateCommunicationType)
				}
			},
			fx.Annotate(
				link_service.NewLinkService,
				fx.As(new(link_service.LinkService)),
			),
			func(cfg *config.Config, logger *slog.Logger) *redis.ClusterClient {
				return cache.NewRedisClient(&cfg.ValKey)
			},
			fx.Annotate(
				func(rdb *redis.ClusterClient, cfg *config.Config, logger *slog.Logger) cache.Cache {
					if !cfg.Scrapper.CacheEnabled {
						logger.Info("cache disabled; using no-cache")
						return cache.NewNoCache()
					}

					return cache.NewValKeyCache(rdb, &cfg.ValKey, "scrapper")
				},
				fx.As(new(cache.Cache)),
			),
			func(cfg *config.Config, rdb *redis.ClusterClient) cache.Invalidator {
				if !cfg.Bot.CacheEnabled {
					return cache.NewNoCacheInvalidator()
				}
				return cache.NewValKeyInvalidator(rdb)
			},
			rest_handlers.NewScrapperRestServer,
			rpc_handlers.NewScrapperRPCServer,
			scrapper.NewScrapper,
			func(
				cfg *config.Config,
				linkRepo link_repository.LinkUpdateRepository,
				logger *slog.Logger,
				updater scheduler.Updater,
				scrapper scrapper.Scrapper,
			) (*scheduler.Scheduler, error) {
				return scheduler.NewScheduler(linkRepo, logger, updater, scrapper, cfg.Scrapper.JobDelayInterval)
			},
		),
		fx.Invoke(run),
	)
}

func main() {
	NewApp().Run()
}
