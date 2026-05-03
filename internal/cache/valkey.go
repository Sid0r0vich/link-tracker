package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
)

type ValKeyCache struct {
	client         redis.Cmdable
	expirationTime time.Duration
	prefix         string
}

func NewRedisClient(cfg *config.ValKeyConfig) *redis.ClusterClient {
	return redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    cfg.Addrs,
		Username: cfg.User,
		Password: cfg.Password,
	})
}

func NewValKeyCache(client redis.Cmdable, cfg *config.ValKeyConfig, prefix string) *ValKeyCache {
	return &ValKeyCache{client: client, expirationTime: cfg.ExpirationTime, prefix: prefix}
}

func (c *ValKeyCache) idToStr(chatID int64) string {
	return fmt.Sprintf("%s:%d", c.prefix, chatID)
}

func (c *ValKeyCache) Get(chatID int64) ([]byte, error) {
	value, err := c.client.Get(context.Background(), c.idToStr(chatID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	return []byte(value), nil
}

func (c *ValKeyCache) Set(chatID int64, data []byte) error {
	return c.client.Set(context.Background(), c.idToStr(chatID), data, c.expirationTime).Err()
}

func (c *ValKeyCache) Delete(chatID int64) error {
	return c.client.Del(context.Background(), c.idToStr(chatID)).Err()
}

type ValKeyInvalidator struct {
	client *redis.ClusterClient
}

func NewValKeyInvalidator(client *redis.ClusterClient) *ValKeyInvalidator {
	return &ValKeyInvalidator{client: client}
}

func (i *ValKeyInvalidator) Invalidate(chatID int64) error {
	return i.client.Publish(context.Background(), "invalidate", chatID).Err()
}
