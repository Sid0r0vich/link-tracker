package main

import (
	"context"
	"fmt"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/broker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	brokerhandler "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/handlers/broker"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/logs"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/update"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
	"go.uber.org/fx"
)

func run(
	ctx context.Context,
	cfg *config.Config,
	logger *slog.Logger,
	handler *brokerhandler.AgentMessageHandler,
) error {
	if err := broker.StartConsumerGroup(ctx, broker.NewConfig(), cfg.Kafka.Brokers, cfg.Kafka.Raw.Topic, cfg.Kafka.Raw.GroupID, logger, handler.Handle); err != nil {
		return fmt.Errorf("start consumer group: %w", err)
	}

	return nil
}

func NewApp() *fx.App {
	return fx.New(
		fx.Provide(
			utils.GetContext,
			config.LoadConfig,
			logs.NewLogger,
			func(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*update.UpdateBrokerService, error) {
				return update.NewUpdateBrokerService(ctx, cfg.Kafka.Processed.Topic, &cfg.Kafka, logger)
			},
			func(cfg *config.Config, updater *update.UpdateBrokerService, logger *slog.Logger) (*brokerhandler.AgentMessageHandler, error) {
				return brokerhandler.NewAgentMessageHandler(updater, cfg.AIAgent.Filtering, cfg.AIAgent.Summarization, logger)
			},
		),
		fx.Invoke(run),
	)
}

func main() {
	NewApp().Run()
}
