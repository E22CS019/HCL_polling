package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates and verifies a Redis connection.
// It accepts either a full Redis URL (rediss://... or redis://...)
// or separate addr + password values.
func NewRedisClient(redisURL, addr, password string) (*redis.Client, error) {
	var rdb *redis.Client

	if redisURL != "" {
		// Parse the full URL — used by Upstash and other managed Redis providers.
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil, err
		}
		rdb = redis.NewClient(opt)
	} else {
		// Fallback to addr + password — used for local Redis.
		rdb = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       0,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return rdb, nil
}
