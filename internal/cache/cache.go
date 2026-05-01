package cache

import "errors"

var ErrCacheMiss = errors.New("cache miss")

type Cache interface {
	Get(ctxchatID int64) ([]byte, error)
	Set(chatID int64, data []byte) error
	Delete(chatID int64) error
}
