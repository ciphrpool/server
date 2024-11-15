package routes

import (
	"backend/lib/server/middleware"
	"backend/lib/services"
	"context"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/gofiber/fiber/v2"
)

type FriendliesChallengeData struct {
	Opponent_tag string `json:"opponent_tag"`
	Response     bool   `json:"response"`
}

func FriendliesChallengeHandler(data FriendliesChallengeData, ctx *fiber.Ctx, cache *services.Cache, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)
	opponent_id, err := queries.GetUserIDByTag(query_ctx, data.Opponent_tag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot find this user",
		})
	}

	user_ctx, ok := ctx.Locals(middleware.USER_CONTEXT_KEY).(middleware.UserContext)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	// waiting_room_id, err := cache.UpsertDuelWaitingRoom(services.WaitingRoomData{Player1ID: user_ctx.UserID, Player2ID: opponent_id})
	_, err = cache.UpsertDuelWaitingRoom(services.WaitingRoomData{Player1ID: user_ctx.UserID, Player2ID: opponent_id})
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot create arena session",
		})
	}

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot update this relationship status",
		})
	}

	return ctx.SendStatus(fiber.StatusAccepted)
}
