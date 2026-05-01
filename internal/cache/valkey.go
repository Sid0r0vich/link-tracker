package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
)

func idToStr(chatID int64) string {
	return strconv.FormatInt(chatID, 10)
}

type ValKeyCache struct {
	client         *redis.Client
	expirationTime time.Duration
}

func NewValKeyCache(cfg *config.ValKeyConfig) *ValKeyCache {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Username: cfg.User,
		Password: cfg.Password,
	})
	return &ValKeyCache{client: client, expirationTime: cfg.ExpirationTime}
}

func (c *ValKeyCache) Get(chatID int64) ([]byte, error) {
	value, err := c.client.Get(context.Background(), idToStr(chatID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	return []byte(value), nil
}

func (c *ValKeyCache) Set(chatID int64, data []byte) error {
	return c.client.Set(context.Background(), idToStr(chatID), data, c.expirationTime).Err()
}

func (c *ValKeyCache) Delete(chatID int64) error {
	return c.client.Del(context.Background(), idToStr(chatID)).Err()
}
