package config

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient membuat client Redis dengan timeout terbatas agar cache tidak
// menahan request API ketika Redis lambat atau tidak tersedia.
func NewRedisClient(cfg RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:                  cfg.Address,
		Password:              cfg.Password,
		DB:                    cfg.DB,
		DialTimeout:           2 * time.Second,
		ReadTimeout:           time.Second,
		WriteTimeout:          time.Second,
		PoolTimeout:           time.Second,
		ContextTimeoutEnabled: true,
	})
}

// PingRedis memeriksa koneksi Redis dengan batas waktu.
func PingRedis(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}

// CloseRedis menutup koneksi Redis.
func CloseRedis(client *redis.Client) error {
	return client.Close()
}
