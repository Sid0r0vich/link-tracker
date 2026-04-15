package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/db"
	rest_handlers "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rest"
	rpc_handlers "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/rpc"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/middleware"
	link_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
	orm_link_repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link/postgres/orm"
	sql_link_repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link/postgres/sql"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scheduler"
	link_service "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/update"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rest"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper/rpc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

func run(
	cfg *config.Config,
	connCfg *pgxpool.Config,
	logger *slog.Logger,
	sched *scheduler.Scheduler,
	restAPI *rest_handlers.ScrapperRestServer,
	rpcAPI *rpc_handlers.ScrapperRPCServer,
) error {
	logger.Info("start migration")
	if err := db.Migrate(connCfg.ConnConfig); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	sched.Start()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("interrupt signal received. Shutdown")
		sched.Shutdown()
	}()

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

func main() {
	fx.New(
		//fx.NopLogger,
		fx.Provide(
			config.LoadConfig,
			logs.NewLogger,
			db.GetConnCfg,
			fx.Annotate(
				func(
					cfg *config.Config,
					pgxCfg *pgxpool.Config,
					lifecycle fx.Lifecycle,
					logger *slog.Logger,
				) (link_repository.LinkUnitedRepository, error) {
					switch cfg.Scrapper.DBAccessType {
					case "SQL":
						pool, err := db.GetDBPoolConn(pgxCfg)
						if err != nil {
							return nil, fmt.Errorf("connect to db: %w", err)
						}

						lifecycle.Append(fx.Hook{
							OnStop: func(context.Context) error {
								db.CloseDBConn()
								logger.Info("database connection closed")
								return nil
							},
						})

						return sql_link_repo.NewSqlLinkService(pool), nil

					case "ORM":
						db, err := sql.Open("pgx", db.GetDSNFromConfig(cfg))
						if err != nil {
							return nil, fmt.Errorf("fail to open database: %v", err)
						}

						lifecycle.Append(fx.Hook{
							OnStop: func(context.Context) error {
								db.Close()
								logger.Info("database connection closed")
								return nil
							},
						})

						return orm_link_repo.NewORMLinkService(db), nil

					default:
						return nil, fmt.Errorf("invalid db access type: %s", cfg.Scrapper.DBAccessType)
					}
				},
				fx.As(new(link_repository.LinkRepository)),
				fx.As(new(link_repository.LinkUpdateRepository)),
			),
			func(cfg *config.Config) *update.UpdateService {
				return update.NewUpdateService(fmt.Sprintf("http://%s", cfg.Bot.ServerAddr))
			},
			fx.Annotate(
				link_service.NewLinkService,
				fx.As(new(link_service.LinkService)),
			),
			rest_handlers.NewUpdatesRestServer,
			rpc_handlers.NewUpdatesRPCServer,
			func(cfg *config.Config, logger *slog.Logger) scrapper.Scrapper {
				return scrapper.NewScrapperService(map[string]scrapper.Scrapper{
					"api.github.com":        scrapper.NewGithubScrapper(cfg.Scrapper.GithubToken, logger),
					"api.stackexchange.com": scrapper.NewStackoverflowScrapper(cfg.Scrapper.StackoverflowKey),
				})
			},
			func(
				cfg *config.Config,
				linkRepo link_repository.LinkUpdateRepository,
				logger *slog.Logger,
				updater *update.UpdateService,
				scrapper scrapper.Scrapper,
			) (*scheduler.Scheduler, error) {
				return scheduler.NewScheduler(linkRepo, logger, updater, scrapper, time.Duration(cfg.Scrapper.JobDelayIntervalSeconds)*time.Second)
			},
		),
		fx.Invoke(run),
	).Run()
}
