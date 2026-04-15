package config

import (
	"fmt"
	"log/slog"
	"os"

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

type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Bot      BotConfig      `mapstructure:"bot"`
	Scrapper ScrapperConfig `mapstructure:"scrapper"`
}

type DatabaseConfig struct {
	Host                string `mapstructure:"host"`
	Port                int    `mapstructure:"port"`
	Username            string `mapstructure:"username"`
	Password            string `mapstructure:"password"`
	Name                string `mapstructure:"name"`
	MaxConns            int    `mapstructure:"max_conns"`
	MinConns            int    `mapstructure:"min_conns"`
	MaxConnIdleTimeMins int    `mapstructure:"max_conn_idle_time_mins"`
	MaxConnLifeTimeMins int    `mapstructure:"max_conn_life_time_mins"`
}

type BotConfig struct {
	Token      string `mapstructure:"token"`
	ServerAddr string `mapstructure:"server_addr"`
}

type ScrapperConfig struct {
	ServerAddr              string            `mapstructure:"server_addr"`
	GithubToken             string            `mapstructure:"github_token"`
	StackoverflowKey        string            `mapstructure:"stackoverflow_key"`
	DBAccessType            DBAccessType      `mapstructure:"db_access_type"`
	TransportProtocol       TransportProtocol `mapstructure:"transport_protocol"`
	JobDelayIntervalSeconds int               `mapstructure:"job_delay_interval_seconds"`
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

	return cfg, nil
}
