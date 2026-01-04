package usage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/saturnino-fabrica-de-software/rekko/internal/cache"
)

type CacheAdapter struct {
	pgCache *cache.PGCache
}

func NewCacheAdapter(pgCache *cache.PGCache) *CacheAdapter {
	return &CacheAdapter{pgCache: pgCache}
}

func (a *CacheAdapter) Get(ctx context.Context, key string, value interface{}) error {
	data, err := a.pgCache.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, value)
}

func (a *CacheAdapter) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return a.pgCache.Set(ctx, key, data, ttl)
}
