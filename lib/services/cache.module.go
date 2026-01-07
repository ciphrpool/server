package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

func (cache *Cache) RefreshActiveModule(user_id pgtype.UUID, db *Database, force bool) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	module_key := fmt.Sprintf("module:active:%s", UUIDToString(user_id))

	if !force {
		_, err := cache.Db.Get(query_ctx, module_key).Result()
		if err == nil {
			return nil
		} else if err != redis.Nil {
			return fmt.Errorf("failed to get active module from cache: %w", err)
		}
	}
	queries := basepool.New(db.Pool)

	module, err := queries.GetActiveModule(query_ctx, user_id)
	if err != nil {
		return nil // No active module
	}
	module_json, err := json.Marshal(module)
	if err != nil {
		return fmt.Errorf("failed to marshal module: %w", err)
	}
	err = cache.Db.Set(query_ctx, module_key, module_json, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set active module in cache: %w", err)
	}

	return nil
}
