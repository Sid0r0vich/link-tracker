package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/viper"
)

type DBAccessType string

const (
	DBAccessTypeSQL DBAccessType = "SQL"
	DBAccessTypeORM DBAccessType = "ORM"
)

func (d DBAccessType) IsValid() bool {
	switch d {
	case DBAccessTypeSQL, DBAccessTypeORM:
		return true
	default:
		return false
	}
}

type TransportProtocol string

const (
	TransportProtocolHTTP TransportProtocol = "HTTP"
	TransportProtocolGRPC TransportProtocol = "GRPC"
)

func (d TransportProtocol) IsValid() bool {
	switch d {
	case TransportProtocolHTTP, TransportProtocolGRPC:
		return true
	default:
		return false
	}
}

type UpdateCommunicationType string

const (
	UpdateCommunicationTypeKafka    UpdateCommunicationType = "kafka"
	UpdateCommunicationTypeHTTP     UpdateCommunicationType = "HTTP"
	UpdateCommunicationTypeFallback UpdateCommunicationType = "fallback"
)

func (d UpdateCommunicationType) IsValid() bool {
	switch d {
	case UpdateCommunicationTypeKafka, UpdateCommunicationTypeHTTP, UpdateCommunicationTypeFallback:
		return true
	default:
		return false
	}
}

type RetryStrategyType string

const (
	RetryStrategyConstant    RetryStrategyType = "constant"
	RetryStrategyExponential RetryStrategyType = "exponential"
)

func (d RetryStrategyType) IsValid() bool {
	switch d {
	case RetryStrategyConstant, RetryStrategyExponential:
		return true
	default:
		return false
	}
}

type Config struct {
	Database       DatabaseConfig       `mapstructure:"database"`
	Bot            BotConfig            `mapstructure:"bot"`
	Scrapper       ScrapperConfig       `mapstructure:"scrapper"`
	Kafka          KafkaConfig          `mapstructure:"kafka"`
	ValKey         ValKeyConfig         `mapstructure:"valkey"`
	HTTP           HTTPConfig           `mapstructure:"http"`
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

type DatabaseConfig struct {
	Host                  string `mapstructure:"host"`
	Port                  uint16 `mapstructure:"port"`
	Username              string `mapstructure:"username"`
	Password              string `mapstructure:"password"`
	Name                  string `mapstructure:"name"`
	MaxConns              int    `mapstructure:"max_conns"`
	MinConns              int    `mapstructure:"min_conns"`
	MaxConnIdleTimeMins   int    `mapstructure:"max_conn_idle_time_mins"`
	MaxConnLifeTimeMins   int    `mapstructure:"max_conn_life_time_mins"`
	SubscriptionBatchSize uint   `mapstructure:"subscription_batch_size"`
}

type BotConfig struct {
	Token        string `mapstructure:"token"`
	ServerAddr   string `mapstructure:"server_addr"`
	CacheEnabled bool   `mapstructure:"cache_enabled"`
}

type ScrapperConfig struct {
	ServerAddr              string                  `mapstructure:"server_addr"`
	GithubToken             string                  `mapstructure:"github_token"`
	StackoverflowKey        string                  `mapstructure:"stackoverflow_key"`
	DBAccessType            DBAccessType            `mapstructure:"db_access_type"`
	TransportProtocol       TransportProtocol       `mapstructure:"transport_protocol"`
	JobDelayInterval        time.Duration           `mapstructure:"job_delay_interval"`
	UpdateCommunicationType UpdateCommunicationType `mapstructure:"update_communication_type"`
	CacheEnabled            bool                    `mapstructure:"cache_enabled"`
	UrlValidationEnabled    bool                    `mapstructure:"url_validation_enabled"`
	OldUpdatesEnabled       bool                    `mapstructure:"old_updates_enabled"`
}

type KafkaConfig struct {
	Brokers           []string `mapstructure:"brokers"`
	Topic             string   `mapstructure:"topic"`
	GroupID           string   `mapstructure:"group_id"`
	NumPartitions     int32    `mapstructure:"num_partitions"`
	RetentionMs       int      `mapstructure:"retention_ms"`
	MinInsyncReplicas int      `mapstructure:"min_insync_replicas"`
}

type ValKeyConfig struct {
	Addrs          []string      `mapstructure:"addrs"`
	User           string        `mapstructure:"user"`
	Password       string        `mapstructure:"password"`
	ExpirationTime time.Duration `mapstructure:"expiration_time"`
}

type HTTPConfig struct {
	Timeout            time.Duration     `mapstructure:"timeout"`
	RateLimit          int               `mapstructure:"rate_limit"`
	RateLimitInterval  time.Duration     `mapstructure:"rate_limit_interval"`
	RetryCount         uint              `mapstructure:"retry_count"`
	RetryDelay         time.Duration     `mapstructure:"retry_delay"`
	RetryableHTTPCodes []int             `mapstructure:"retryable_http_codes"`
	RetryStrategy      RetryStrategyType `mapstructure:"retry_strategy"`
}

type CircuitBreakerConfig struct {
	SlidingWindowSize        time.Duration `mapstructure:"sliding_window_size"`
	SlidingWindowBucketSize  time.Duration `mapstructure:"sliding_window_bucket_size"`
	MinimumRequiredCalls     uint32        `mapstructure:"minimum_required_calls"`
	FailureRateThreshold     float64       `mapstructure:"failure_rate_threshold"`
	PermittedCallsInHalfOpen uint32        `mapstructure:"permitted_calls_in_half_open_state"`
	WaitDurationInOpenState  time.Duration `mapstructure:"wait_duration_in_open_state"`
}

func LoadConfig(logger *slog.Logger) (*Config, error) {
	logger.Info("load config")

	configFileName := os.Getenv("CONFIG_FILE")
	cfg, err := newConfigFromFile(configFileName)
	if err != nil {
		return nil, fmt.Errorf("load config from file: %w", err)
	}

	return cfg, nil
}

func newConfigFromFile(name string) (*Config, error) {
	cfg := &Config{}

	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(name)
	v.AutomaticEnv()

	v.BindEnv("database.username", "POSTGRES_USER")
	v.BindEnv("database.password", "POSTGRES_PASSWORD")
	v.BindEnv("database.name", "POSTGRES_DB")

	v.BindEnv("valkey.user", "VALKEY_USER")
	v.BindEnv("valkey.password", "VALKEY_PASSWORD")

	v.BindEnv("bot.token", "BOT_TOKEN")

	v.BindEnv("scrapper.db_access_type", "SCRAPPER_DB_ACCESS_TYPE")
	v.BindEnv("scrapper.transport_protocol", "SCRAPPER_TRANSPORT_PROTOCOL")
	v.BindEnv("scrapper.job_delay_interval", "SCRAPPER_JOB_DELAY_INTERVAL")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if !cfg.Scrapper.DBAccessType.IsValid() {
		return nil, fmt.Errorf("invalid db access type: %s", cfg.Scrapper.DBAccessType)
	}

	if !cfg.Scrapper.TransportProtocol.IsValid() {
		return nil, fmt.Errorf("invalid transport protocol: %s", cfg.Scrapper.TransportProtocol)
	}

	if !cfg.Scrapper.UpdateCommunicationType.IsValid() {
		return nil, fmt.Errorf("invalid update communication type: %s", cfg.Scrapper.UpdateCommunicationType)
	}

	if !cfg.HTTP.RetryStrategy.IsValid() {
		return nil, fmt.Errorf("invalid retry strategy: %s", cfg.HTTP.RetryStrategy)
	}

	return cfg, nil
}
