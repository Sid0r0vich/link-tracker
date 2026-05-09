package ratelimiter

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

type Ratelimiter struct {
	data   *cache.Cache
	limit  int
	prefix string
}

func New(limit int, expirationTime time.Duration, prefix string) *Ratelimiter {
	return &Ratelimiter{
		data:   cache.New(expirationTime, expirationTime/2),
		limit:  limit,
		prefix: prefix,
	}
}

func (r *Ratelimiter) ipToStr(ip string) string {
	return fmt.Sprintf("%s:%s", r.prefix, ip)
}

func (r *Ratelimiter) Limit(ip string) (bool, error) {
	var limiter *rate.Limiter

	limiterStr, found := r.data.Get(r.ipToStr(ip))
	if !found {
		limiter = rate.NewLimiter(1, r.limit)
		r.data.Set(r.ipToStr(ip), limiter, cache.DefaultExpiration)
	} else {
		var ok bool
		limiter, ok = limiterStr.(*rate.Limiter)
		if !ok {
			return false, fmt.Errorf("failed to assert limiter type")
		}
	}

	return limiter.Allow(), nil
}
