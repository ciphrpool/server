package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	Db *redis.Client
}

func DefaultCache() Cache {
	return Cache{
		Db: nil,
	}
}

func (cache *Cache) Connect(password string) error {
	address := os.Getenv("CACHE_ADDRESS")
	db := redis.NewClient(&redis.Options{
		Addr:     address,
		Username: "MCS",
		Password: password,
	})
	cache.Db = db
	ctx := context.Background()
	_, err := cache.Db.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Dragonfly: %w", err)
	}
	slog.Info("Cache connection succeeded")
	return nil
}
