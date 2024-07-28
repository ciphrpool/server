package database

import (
	"backend/lib"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Cache struct {
	Db *redis.Client
}

func NewCache() (Cache, error) {
	return Cache{
		Db: nil,
	}, nil
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

func (cache *Cache) AddEngine(engine lib.Engine) error {
	if engine.Id == "" {
		engine.Id = uuid.New().String()
	}
	ctx := context.Background()
	engine_json, err := json.Marshal(engine)
	if err != nil {
		return fmt.Errorf("failed to marshal engine: %w", err)
	}

	err = cache.Db.Set(ctx, fmt.Sprintf("engine:%s", engine.Id), engine_json, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to add engine: %w", err)
	}
	return nil
}

func (cache *Cache) GetEngine(id string) (lib.Engine, error) {
	var engine lib.Engine
	ctx := context.Background()
	engine_json, err := cache.Db.Get(ctx, fmt.Sprintf("engine:%s", id)).Result()
	if err == redis.Nil {
		return engine, fmt.Errorf("engine with ID %s does not exist", id)
	} else if err != nil {
		return engine, fmt.Errorf("failed to get engine: %w", err)
	}
	if err := json.Unmarshal([]byte(engine_json), &engine); err != nil {
		return engine, fmt.Errorf("failed to unmarshal engine data: %w", err)
	}
	return engine, nil
}

func (cache *Cache) UpdateEngine(engine lib.Engine) error {
	if engine.Id == "" {
		return fmt.Errorf("engine ID cannot be empty")
	}
	ctx := context.Background()
	engine_json, err := json.Marshal(engine)
	if err != nil {
		return fmt.Errorf("failed to marshal engine: %w", err)
	}
	err = cache.Db.Set(ctx, fmt.Sprintf("engine:%s", engine.Id), engine_json, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to update engine: %w", err)
	}
	return nil
}
