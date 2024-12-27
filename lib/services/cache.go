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
