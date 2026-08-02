// Package cache wraps Redis for the two things the API needs from it: a
// sliding-window request counter for rate limiting, and a JWT blacklist for
// immediate logout invalidation.
package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"pilot-backend/internal/config"
)

const blacklistKeyPrefix = "blacklist:"

// Redis is a thin wrapper around a go-redis client.
type Redis struct {
	client *redis.Client
}

// NewRedis builds a Redis cache client from cfg. It does not eagerly
// connect — the first command establishes the connection.
func NewRedis(cfg *config.Config) *Redis {
	return &Redis{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.RedisHost + ":" + strconv.Itoa(cfg.RedisPort),
			Password: cfg.RedisPassword,
		}),
	}
}

// Ping verifies connectivity to Redis.
func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close releases the underlying connection pool.
func (r *Redis) Close() error {
	return r.client.Close()
}

// Incr increments key and returns its new value, creating it at 1 if it
// didn't exist.
func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Expire sets a TTL on key.
func (r *Redis) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

// Blacklist marks token as invalid for ttl (normally the token's remaining
// lifetime), so ValidateToken rejects it even though its signature and
// expiry are still otherwise valid.
func (r *Redis) Blacklist(ctx context.Context, token string, ttl time.Duration) error {
	return r.client.Set(ctx, blacklistKeyPrefix+token, "1", ttl).Err()
}

// IsBlacklisted reports whether token has been blacklisted.
func (r *Redis) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	n, err := r.client.Exists(ctx, blacklistKeyPrefix+token).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
