package broker

import (
	"github.com/IBM/sarama"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
)

func NewConfig(appCfg *config.Config, opts ...func(*sarama.Config)) *sarama.Config {
	cfg := sarama.NewConfig()

	cfg.Version = sarama.V4_0_0_0
	cfg.Producer.Partitioner = sarama.NewHashPartitioner
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Compression = sarama.CompressionGZIP

	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	cfg.Consumer.Offsets.AutoCommit.Enable = false

	for _, o := range opts {
		o(cfg)
	}

	return cfg
}
