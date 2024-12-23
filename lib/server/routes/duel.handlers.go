package routes

import (
	"backend/lib/notifications"
	"backend/lib/server/middleware"
	"backend/lib/server/routes/security"
	"backend/lib/services"
	"backend/lib/vault"
	"context"
	"encoding/json"
	"fmt"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/gofiber/fiber/v2"
)

type FriendliesChallengeData struct {
	Opponent_tag string `json:"opponent_tag"`
}

func FriendliesChallengeHandler(data FriendliesChallengeData, ctx *fiber.Ctx, cache *services.Cache, db *services.Database, notify *notifications.NotificationService) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)
	opponent, err := queries.GetUserIDByTag(query_ctx, data.Opponent_tag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot find this user",
		})
	}

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	// waiting_room_id, err := cache.UpsertDuelWaitingRoom(services.WaitingRoomData{Player1ID: user_id, Player2ID: opponent_id})
	_, is_new, err := cache.UpsertDuelWaitingRoom(services.WaitingRoomData{Player1ID: user_id, Player2ID: opponent.ID})
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot create arena session",
		})
	}
	// Notify opponent
	if is_new {
		notify.Send(
			ctx.Context(),
			notifications.TypeAlert,
			"duel:challenge_notification",
			notifications.PriorityHigh,
			opponent.ID,
			fiber.Map{
				"msg": fmt.Sprintf("%s#%s has challenged you to a friendly duel !", opponent.Username, opponent.Tag),
			},
			fiber.Map{
				"opponent_tag": opponent.Tag,
				"expired_in":   services.WAITING_ROOM_TTL,
			},
		)
	}
	return ctx.SendStatus(fiber.StatusAccepted)
}

type FriendliesChallengeResponseData struct {
	Opponent_tag string `json:"opponent_tag"`
	Response     bool   `json:"response"`
}

func FriendliesChallengeResponeHandler(data FriendliesChallengeResponseData, ctx *fiber.Ctx, cache *services.Cache, db *services.Database, vault *vault.VaultManager, notify *notifications.NotificationService) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	p1_duel_summary_data, err := queries.GetUserDuelSummaryDataByTag(query_ctx, data.Opponent_tag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot find this user",
		})
	}

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	if !data.Response {
		// The duel is rejected
		err := cache.DeleteDuelWaitingRoom(services.WaitingRoomData{Player1ID: p1_duel_summary_data.ID, Player2ID: user_id})
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot delete friendly duel waiting room",
			})
		}
		return ctx.SendStatus(fiber.StatusAccepted)
	}

	// The duel is accepted

	p2_duel_summary_data, err := queries.GetUserDuelSummaryDataById(query_ctx, user_id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot find this user",
		})
	}

	session_id, err := cache.CreateDuelSession(&services.DuelSessionData{
		DuelType: basepool.DuelTypeFriendly,
		P1: services.DuelPlayerSummaryData{
			PID:      p1_duel_summary_data.ID,
			Elo:      uint(p1_duel_summary_data.Elo),
			Tag:      p1_duel_summary_data.Tag,
			Username: p1_duel_summary_data.Username,
		},
		P2: services.DuelPlayerSummaryData{
			PID:      p2_duel_summary_data.ID,
			Elo:      uint(p2_duel_summary_data.Elo),
			Tag:      p2_duel_summary_data.Tag,
			Username: p2_duel_summary_data.Tag,
		},
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot create duel session",
		})
	}

	aes_key := vault.OpenNexusAESKey

	p1_session_json, err := json.Marshal(fiber.Map{
		"session_id": session_id,
		"pid":        p1_duel_summary_data.ID,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot start the game",
		})
	}

	p2_session_json, err := json.Marshal(fiber.Map{
		"session_id": session_id,
		"pid":        p2_duel_summary_data.ID,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot start the game",
		})
	}
	// p1_crypted_session_payload, err := security.EncryptAES(string(p1_session_json[:]), aes_key)
	_, err = security.EncryptAES(string(p1_session_json[:]), aes_key)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot start the game",
		})
	}
	// p2_crypted_session_payload, err := security.EncryptAES(string(p2_session_json[:]), aes_key)
	_, err = security.EncryptAES(string(p2_session_json[:]), aes_key)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot start the game",
		})
	}
	// Notify both players with the Duel Session
	notify.Send(
		ctx.Context(),
		notifications.TypeRedirect,
		"duel:acceptance",
		notifications.PriorityHigh,
		p1_duel_summary_data.ID,
		fiber.Map{
			"msg": "The friendly duel has been accepted, you will be redirected to the duel...",
		},
		fiber.Map{
			"url": "",
		},
	)
	notify.Send(
		ctx.Context(),
		notifications.TypeRedirect,
		"duel:acceptance",
		notifications.PriorityHigh,
		user_id,
		fiber.Map{
			"msg": "The friendly duel has been accepted, you will be redirected to the duel...",
		},
		fiber.Map{
			"url": "",
		},
	)
	err = cache.DeleteDuelWaitingRoom(services.WaitingRoomData{Player1ID: p1_duel_summary_data.ID, Player2ID: user_id})
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot create arena session",
		})
	}

	return ctx.SendStatus(fiber.StatusAccepted)
}
