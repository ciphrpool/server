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
	"strings"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
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
	user, err := queries.GetUserByID(query_ctx, user_id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot find this user",
		})
	}

	waiting_room_id, err := cache.UpsertDuelWaitingRoom(services.WaitingRoomData{Player1ID: user_id, Player2ID: opponent.ID})
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot create this duel waiting room",
		})
	}
	// Notify opponent
	notify.Send(
		ctx.Context(),
		notifications.TypeAlert,
		"duel:challenge:request",
		notifications.PriorityHigh,
		opponent.ID,
		fiber.Map{
			"msg": fmt.Sprintf("%s#%s has challenged you to a friendly duel !", user.Username, user.Tag),
		},
		fiber.Map{
			"waiting_room_id": waiting_room_id,
			"opponent_tag":    user.Tag,
			"expired_at":      time.Now().Add(services.WAITING_ROOM_TTL).UnixMilli(),
		},
	)
	// Notify user of duel creation
	notify.Send(
		ctx.Context(),
		notifications.TypeAlert,
		"duel:challenge:creation",
		notifications.PriorityHigh,
		user_id,
		fiber.Map{
			"msg": fmt.Sprintf("Friendly duel against %s#%s will start soon", opponent.Username, opponent.Tag),
		},
		fiber.Map{
			"waiting_room_id": waiting_room_id,
			"opponent_tag":    opponent.Tag,
			"expired_at":      time.Now().Add(services.WAITING_ROOM_TTL).UnixMilli(),
		},
	)
	return ctx.SendStatus(fiber.StatusOK)
}

type FriendliesChallengeResponseData struct {
	WaitingRoomId string `json:"waiting_room_id"`
	Opponent_tag  string `json:"opponent_tag"`
	Response      bool   `json:"response"`
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

	waiting_room_id, err := cache.GetDuelWaitingRoom(services.WaitingRoomData{Player1ID: p1_duel_summary_data.ID, Player2ID: user_id})
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No waiting room found",
		})
	}
	if waiting_room_id != data.WaitingRoomId {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid waiting room",
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
		notify.Send(
			ctx.Context(),
			notifications.TypeRedirect,
			"duel:acceptance:rejection",
			notifications.PriorityHigh,
			p1_duel_summary_data.ID,
			fiber.Map{
				"msg": "The friendly duel has been rejected",
			},
			fiber.Map{},
		)
		return ctx.SendStatus(fiber.StatusOK)
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
			Username: p2_duel_summary_data.Username,
		},
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot create duel session",
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
			"duel_session_id": session_id,
			"duel_type":       "friendly",
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
			"duel_session_id": session_id,
			"duel_type":       "friendly",
		},
	)
	err = cache.DeleteDuelWaitingRoom(services.WaitingRoomData{Player1ID: p1_duel_summary_data.ID, Player2ID: user_id})
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot create arena session",
		})
	}

	return ctx.SendStatus(fiber.StatusOK)
}

type FriendliesPrepareResponseParams struct {
	DuelSessionId string `query:"duel_session_id"`
}

func FriendliesPrepareHandler(params FriendliesPrepareResponseParams, ctx *fiber.Ctx, cache *services.Cache, db *services.Database, vault *vault.VaultManager, notify *notifications.NotificationService) error {

	session_data, err := cache.GetDuelSession(params.DuelSessionId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "invalid duel session",
		})
	}
	aes_key := vault.OpenNexusAESKey

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}
	var response struct {
		WS_Url                  string `json:"ws_url"`
		SSE_Url                 string `json:"sse_url"`
		EncryptedSessionContext string `json:"encrypted_session_context"`
	}
	nexuspool, err := cache.SearchAliveNexusPool()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No running nexuspool",
		})
	}
	ws_url := nexuspool.Url
	if strings.HasPrefix(ws_url, "http://") {
		ws_url = "ws://" + strings.TrimPrefix(ws_url, "http://")
	} else if strings.HasPrefix(ws_url, "https://") {
		ws_url = "wss://" + strings.TrimPrefix(ws_url, "https://")
	}

	sse_url := nexuspool.Url

	response.WS_Url = fmt.Sprintf("%s/ws/duel/", ws_url)
	response.SSE_Url = fmt.Sprintf("%s/sse/duel/", sse_url)

	if user_id.Bytes == session_data.P1.PID.Bytes {
		p1_session_json, err := json.Marshal(fiber.Map{
			"session_id": params.DuelSessionId,
			"user_id":    session_data.P1.PID,
		})
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to marshal player 1 data",
			})
		}
		p1_crypted_session_payload, err := security.EncryptAESUrlSafe(string(p1_session_json[:]), aes_key)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "cannot start the game",
			})
		}
		response.EncryptedSessionContext = p1_crypted_session_payload

	} else if user_id.Bytes == session_data.P2.PID.Bytes {
		p2_session_json, err := json.Marshal(fiber.Map{
			"session_id": params.DuelSessionId,
			"user_id":    session_data.P2.PID,
		})
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to marshal player 2 data",
			})
		}

		p2_crypted_session_payload, err := security.EncryptAESUrlSafe(string(p2_session_json[:]), aes_key)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "cannot start the game",
			})
		}
		response.EncryptedSessionContext = p2_crypted_session_payload
	} else {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid user",
		})
	}

	err = cache.RefreshActiveModule(user_id, db)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(response)
}

type GetDuelSessionDataParams struct {
	DuelSessionId string `query:"duel_session_id"`
}

func GetDuelSessionDataHandler(params GetDuelSessionDataParams, ctx *fiber.Ctx, cache *services.Cache, db *services.Database, vault *vault.VaultManager, notify *notifications.NotificationService) error {

	session_data, err := cache.GetDuelSession(params.DuelSessionId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "invalid duel session",
		})
	}
	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}
	var response struct {
		Context services.DuelSessionDataExtern `json:"context"`
		Side    int                            `json:"side"`
	}
	response.Context = services.DuelSessionDataExtern{
		P1: services.DuelPlayerSummaryDataExtern{
			Elo:      session_data.P1.Elo,
			Tag:      session_data.P1.Tag,
			Username: session_data.P1.Username,
		},
		P2: services.DuelPlayerSummaryDataExtern{
			Elo:      session_data.P2.Elo,
			Tag:      session_data.P2.Tag,
			Username: session_data.P2.Username,
		},
		DuelType: session_data.DuelType,
	}

	if user_id.Bytes == session_data.P1.PID.Bytes {
		response.Side = 1
	} else if user_id.Bytes == session_data.P2.PID.Bytes {
		response.Side = 2
	} else {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid user",
		})
	}

	return ctx.JSON(response)
}

type GetDuelResultParams struct {
	DuelSessionId string `query:"duel_session_id"`
}

func GetDuelResultHandler(params GetDuelResultParams, ctx *fiber.Ctx, cache *services.Cache, db *services.Database, vault *vault.VaultManager, notify *notifications.NotificationService) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	sessionID, err := services.StringToUUID(params.DuelSessionId)
	if err != nil {
		return fmt.Errorf("failed to conevrt session id: %w", err)
	}
	duel_result, err := queries.GetDuelResult(query_ctx, basepool.GetDuelResultParams{
		UserID:    user_id,
		SessionID: sessionID,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "invalid duel session",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"result": duel_result,
	})
}

type GetDuelHistoryParams struct {
	Limit        int    `query:"limit"`
	Cursor       string `query:"cursor"`
	Opponent_tag string `query:"opponent_tag"`
}

func GetDuelHistoryHandler(params GetDuelHistoryParams, ctx *fiber.Ctx, cache *services.Cache, db *services.Database, vault *vault.VaultManager, notify *notifications.NotificationService) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}
	if params.Limit == 0 {
		params.Limit = 20 // Default limit
	}
	var cursor pgtype.Timestamptz
	if params.Cursor != "" {
		cursor, err = services.StringToTimestampz(params.Cursor)
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid cursor",
			})
		}
	}

	if params.Opponent_tag != "" {
		opponent, err := queries.GetUserIDByTag(query_ctx, params.Opponent_tag)
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "cannot find this user",
			})
		}

		duels, err := queries.GetUserDuelHistoryFilterTag(query_ctx, basepool.GetUserDuelHistoryFilterTagParams{
			UserID:     user_id,
			TimeCursor: cursor,
			LimitAt:    int32(params.Limit + 1), // Get one extra to check if there are more results
			OppID:      opponent.ID,
		})
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to fetch duel history",
			})
		}

		hasMore := len(duels) > params.Limit
		if hasMore {
			duels = duels[:params.Limit] // Remove the extra item
		}

		nextCursor := ""
		if hasMore && len(duels) > 0 {
			nextCursor = duels[len(duels)-1].Date.Time.Format(time.RFC3339)
		}

		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
			"duels":      duels,
			"hasMore":    hasMore,
			"nextCursor": nextCursor,
		})
	} else {
		duels, err := queries.GetUserDuelHistory(query_ctx, basepool.GetUserDuelHistoryParams{
			UserID:     user_id,
			TimeCursor: cursor,
			LimitAt:    int32(params.Limit + 1), // Get one extra to check if there are more results
		})
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to fetch duel history",
			})
		}

		hasMore := len(duels) > params.Limit
		if hasMore {
			duels = duels[:params.Limit] // Remove the extra item
		}

		nextCursor := ""
		if hasMore && len(duels) > 0 {
			nextCursor = duels[len(duels)-1].Date.Time.Format(time.RFC3339)
		}

		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
			"duels":      duels,
			"hasMore":    hasMore,
			"nextCursor": nextCursor,
		})
	}
}
