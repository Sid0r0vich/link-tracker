package cache_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	tcvalkey "github.com/testcontainers/testcontainers-go/modules/valkey"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/cache"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
)

type ValkeyTestSuite struct {
	suite.Suite
	tc             *tcvalkey.ValkeyContainer
	cache          *cache.ValKeyCache
	expirationTime time.Duration
}

func (s *ValkeyTestSuite) SetupSuite() {
	ctx := context.Background()

	var err error
	s.tc, err = tcvalkey.Run(
		ctx,
		"valkey/valkey:latest",
	)
	require.NoError(s.T(), err, "failed to start valkey container")

	port, err := s.tc.MappedPort(ctx, "6379")
	s.Require().NoError(err)

	s.expirationTime = time.Second
	cfg := &config.ValKeyConfig{
		Addr:           fmt.Sprintf("localhost:%s", port.Port()),
		ExpirationTime: s.expirationTime,
	}
	s.cache = cache.NewValKeyCache(
		cache.NewRedisClient(cfg),
		cfg,
		"test",
	)
}

func (s *ValkeyTestSuite) TearDownSuite() {
	if err := s.tc.Terminate(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to terminate container: %v\n", err)
	}
}

func TestValkeyTestSuite(t *testing.T) {
	suite.Run(t, new(ValkeyTestSuite))
}

func (s *ValkeyTestSuite) TestValkeyGetSetDelete() {
	chatID := int64(12345)
	data := []byte("test data")

	_, err := s.cache.Get(chatID)
	require.ErrorIs(s.T(), err, cache.ErrCacheMiss)

	s.Require().NoError(s.cache.Set(chatID, data))
	value, err := s.cache.Get(chatID)
	s.Require().NoError(err)
	s.Require().Equal(data, value)

	s.Require().NoError(s.cache.Delete(chatID))
	_, err = s.cache.Get(chatID)
	s.Require().ErrorIs(err, cache.ErrCacheMiss)

	s.Require().NoError(s.cache.Set(chatID, data))
	time.Sleep(s.expirationTime + time.Millisecond)
	_, err = s.cache.Get(chatID)
	s.Require().ErrorIs(err, cache.ErrCacheMiss)
}
