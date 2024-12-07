package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

const WAITING_ROOM_TTL = 10 * time.Minute

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

func (cache *Cache) UpsertArenaUnregisteredSession(client_ip string) (string, error) {
	ctx := context.Background()

	session_key := fmt.Sprintf("arena:session:unregistered:%s", client_ip)
	slog.Debug("Areana Session key", "session_key", session_key)
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

type WaitingRoomData struct {
	Player1ID pgtype.UUID `json:"player_1"`
	Player2ID pgtype.UUID `json:"player_2"`
}

func (cache *Cache) UpsertDuelWaitingRoom(waiting_room WaitingRoomData) (string, bool, error) {
	ctx := context.Background()

	waiting_room_key := fmt.Sprintf("duel:waiting_room:id:%s-%s", UUIDToString(waiting_room.Player1ID), UUIDToString(waiting_room.Player2ID))
	waiting_room_id, err := cache.Db.Get(ctx, waiting_room_key).Result()
	if err == nil {
		return waiting_room_id, false, nil
	} else if err != redis.Nil {
		// An error occurred other than key not existing
		return "", true, fmt.Errorf("failed to check existing waiting_room: %w", err)
	}

	waiting_room_id = uuid.New().String()

	err = cache.Db.Set(ctx, waiting_room_key, waiting_room_id, WAITING_ROOM_TTL).Err()
	if err != nil {
		return "", true, fmt.Errorf("failed to create arena session: %w", err)
	}

	return waiting_room_id, true, nil
}

func (cache *Cache) DeleteDuelWaitingRoom(waiting_room WaitingRoomData) error {
	ctx := context.Background()

	waiting_room_key := fmt.Sprintf("duel:waiting_room:id:%s-%s", UUIDToString(waiting_room.Player1ID), UUIDToString(waiting_room.Player2ID))
	_, err := cache.Db.Del(ctx, waiting_room_key).Result()
	if err == nil {
		return nil
	} else if err != redis.Nil {
		// An error occurred other than key not existing
		return fmt.Errorf("failed to check existing waiting_room: %w", err)
	}
	return nil
}

type DuelPlayerSummaryData struct {
	PID      pgtype.UUID `json:"pid"`
	Elo      uint        `json:"elo"`
	Tag      string      `json:"tag"`
	Username string      `json:"username"`
}

type DuelSessionData struct {
	P1       DuelPlayerSummaryData `json:"p1"`
	P2       DuelPlayerSummaryData `json:"p2"`
	DuelType basepool.DuelType
}

func (cache *Cache) CreateDuelSession(session_data *DuelSessionData) (string, error) {
	ctx := context.Background()

	duel_session_id := uuid.New().String()

	duel_session_key := fmt.Sprintf("duel:session:%s", duel_session_id)

	session_data_json, err := json.Marshal(session_data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal duel session data: %w", err)
	}
	err = cache.Db.Set(ctx, duel_session_key, session_data_json, 1*time.Hour).Err()
	if err != nil {
		return "", fmt.Errorf("failed to create duel session cache data: %w", err)
	}

	return duel_session_id, nil
}
