package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type NexusPool struct {
	Id    string `json:"id"`
	Alive bool   `json:"alive"`
	Url   string `json:"url"`
}

func (cache *Cache) AddNexusPool(nexuspool NexusPool) error {
	if nexuspool.Id == "" {
		nexuspool.Id = uuid.New().String()
	}
	ctx := context.Background()
	nexuspool_json, err := json.Marshal(nexuspool)
	if err != nil {
		return fmt.Errorf("failed to marshal nexuspool: %w", err)
	}

	err = cache.Db.Set(ctx, fmt.Sprintf("nexuspool:%s", nexuspool.Id), nexuspool_json, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to add nexuspool: %w", err)
	}
	return nil
}

func (cache *Cache) GetNexusPool(id string) (NexusPool, error) {
	var nexuspool NexusPool
	ctx := context.Background()
	nexuspool_json, err := cache.Db.Get(ctx, fmt.Sprintf("nexuspool:%s", id)).Result()
	if err == redis.Nil {
		return nexuspool, fmt.Errorf("nexuspool with ID %s does not exist", id)
	} else if err != nil {
		return nexuspool, fmt.Errorf("failed to get nexuspool: %w", err)
	}
	if err := json.Unmarshal([]byte(nexuspool_json), &nexuspool); err != nil {
		return nexuspool, fmt.Errorf("failed to unmarshal nexuspool data: %w", err)
	}
	return nexuspool, nil
}

func (cache *Cache) GetAliveNexusPool() (NexusPool, error) {
	var nexuspool NexusPool
	ctx := context.Background()

	// Scan for all nexuspool keys
	var cursor uint64 = 0
	for {
		keys, newCursor, err := cache.Db.Scan(ctx, cursor, "nexuspool:*", 100).Result()
		if err != nil {
			return nexuspool, fmt.Errorf("failed to scan nexuspools: %w", err)
		}

		// Check each pool
		for _, key := range keys {
			nexuspool_json, err := cache.Db.Get(ctx, key).Result()
			if err != nil {
				continue // Skip if we can't get this pool
			}

			if err := json.Unmarshal([]byte(nexuspool_json), &nexuspool); err != nil {
				continue // Skip if we can't unmarshal
			}

			// Return the first alive pool we find
			if nexuspool.Alive {
				return nexuspool, nil
			}
		}

		// Exit if we've scanned all keys
		if newCursor == 0 {
			break
		}
		cursor = newCursor
	}

	return nexuspool, fmt.Errorf("no alive nexuspool found")
}

func (cache *Cache) UpdateNexusPool(nexuspool NexusPool) error {
	if nexuspool.Id == "" {
		return fmt.Errorf("nexuspool ID cannot be empty")
	}
	ctx := context.Background()
	nexuspool_json, err := json.Marshal(nexuspool)
	if err != nil {
		return fmt.Errorf("failed to marshal nexuspool: %w", err)
	}
	err = cache.Db.Set(ctx, fmt.Sprintf("nexuspool:%s", nexuspool.Id), nexuspool_json, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to update nexuspool: %w", err)
	}
	return nil
}
func (cache *Cache) SearchAliveNexusPool() (NexusPool, error) {
	ctx := context.Background()

	// Use SCAN to iterate through all keys
	var cursor uint64
	var nexuspool NexusPool

	for {
		var keys []string
		var err error
		keys, cursor, err = cache.Db.Scan(ctx, cursor, "nexuspool:*", 10).Result()
		if err != nil {
			return nexuspool, fmt.Errorf("failed to scan keys: %w", err)
		}

		for _, key := range keys {
			nexuspool_json, err := cache.Db.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			if err := json.Unmarshal([]byte(nexuspool_json), &nexuspool); err != nil {
				continue
			}

			if nexuspool.Alive {
				return nexuspool, nil
			}
		}

		if cursor == 0 {
			break
		}
	}

	// If no alive nexuspool is found
	return nexuspool, fmt.Errorf("no alive nexuspool found")
}
