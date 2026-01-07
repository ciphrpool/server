package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (cache *Cache) UpsertArenaSession(user_id string) (string, error) {
	ctx := context.Background()

	session_key := fmt.Sprintf("arena:session:%s", user_id)
	// slog.Debug("Areana Session key", "session_key", session_key)

	existing_session_id, err := cache.Db.Get(ctx, session_key).Result()
	if err == nil {
		return existing_session_id, nil
	} else if err != redis.Nil {
		// An error occurred other than key not existing
		return "", fmt.Errorf("failed to check existing session: %w", err)
	}

	session_id := uuid.New().String()

	err = cache.Db.Set(ctx, session_key, session_id, WAITING_ROOM_TTL).Err()
	if err != nil {
		return "", fmt.Errorf("failed to create arena session: %w", err)
	}

	return session_id, nil
}
