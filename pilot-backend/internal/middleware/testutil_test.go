package middleware

import (
	"testing"

	"pilot-backend/internal/config"
	"pilot-backend/pkg/cache"
)

// testRedis connects to the Redis instance described by the environment
// (see docker-compose.yml). It skips the calling test when unreachable.
func testRedis(t *testing.T) *cache.Redis {
	t.Helper()

	cfg := config.Load()
	redisCache := cache.NewRedis(cfg)

	if err := redisCache.Ping(t.Context()); err != nil {
		t.Skipf("redis not reachable, skipping middleware test: %v", err)
	}
	t.Cleanup(func() { redisCache.Close() })

	return redisCache
}
