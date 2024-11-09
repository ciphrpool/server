package services

import (
	"backend/lib"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

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
	cache.StartPeriodicCleanups()
	return nil
}

func (cache *Cache) AddEngine(nexuspool lib.NexusPool) error {
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

func (cache *Cache) GetEngine(id string) (lib.NexusPool, error) {
	var nexuspool lib.NexusPool
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

func (cache *Cache) UpdateEngine(nexuspool lib.NexusPool) error {
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
func (cache *Cache) SearchAliveEngine() (lib.NexusPool, error) {
	ctx := context.Background()

	// Use SCAN to iterate through all keys
	var cursor uint64
	var nexuspool lib.NexusPool

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

func (cache *Cache) UpsertArenaSession(client_ip string) (string, error) {
	ctx := context.Background()

	session_key := fmt.Sprintf("arena:session:ip:%s", client_ip)
	existing_session_id, err := cache.Db.Get(ctx, session_key).Result()
	if err == nil {
		return existing_session_id, nil
	} else if err != redis.Nil {
		// An error occurred other than key not existing
		return "", fmt.Errorf("failed to check existing session: %w", err)
	}

	session_id := uuid.New().String()

	session := lib.ArenaSession{
		ConnexionExpirationDate: time.Now().Add(15 * time.Minute),
		UserID:                  "",
		Alive:                   true,
		Started:                 false,
		UserIP:                  client_ip,
	}

	session_json, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("failed to marshal arena session: %w", err)
	}
	err = cache.Db.Set(ctx, fmt.Sprintf("arena:session:%s", session_id), session_json, 0).Err()
	if err != nil {
		return "", fmt.Errorf("failed to create arena session: %w", err)
	}
	err = cache.Db.Set(ctx, fmt.Sprintf("arena:session:ip:%s", session_id), session_id, 0).Err()
	if err != nil {
		return "", fmt.Errorf("failed to create arena session: %w", err)
	}

	return session_id, nil
}

func (cache *Cache) cleanupExpiredArenaSessions() error {
	ctx := context.Background()
	var cursor uint64

	for {
		var keys []string
		var err error
		keys, cursor, err = cache.Db.Scan(ctx, cursor, "arena:session:*", 100).Result()

		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		for _, key := range keys {
			session_json, err := cache.Db.Get(ctx, key).Result()
			if err != nil {
				continue // Skip this key if there's an error
			}

			var session lib.ArenaSession
			if err := json.Unmarshal([]byte(session_json), &session); err != nil {
				continue // Skip this key if unmarshaling fails
			}

			if !session.Alive {
				if err := cache.Db.Del(ctx, key).Err(); err != nil {
					slog.Error("failed to delete expired session", "key", key, "error", err)
				}
				if err := cache.Db.Del(ctx, fmt.Sprintf("arena:session:ip:%s", session.UserIP)).Err(); err != nil {
					slog.Error("failed to delete expired session related ip", "key", key, "error", err)
				}
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

func (cache *Cache) StartPeriodicCleanups() {
	// Clean ups arena session
	arena_cleanups_interval := 1 * time.Hour
	go func() {
		ticker := time.NewTicker(arena_cleanups_interval)
		defer ticker.Stop()

		for {
			<-ticker.C
			if err := cache.cleanupExpiredArenaSessions(); err != nil {
				slog.Error("failed to cleanup expired sessions", "error", err)
			}
		}
	}()
}
