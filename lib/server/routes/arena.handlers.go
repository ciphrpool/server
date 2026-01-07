package routes

import (
	"backend/lib/server/middleware"
	"backend/lib/server/routes/security"
	"backend/lib/services"
	"backend/lib/vault"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func PrepareArenaHandler(ctx *fiber.Ctx, cache *services.Cache, db *services.Database, vault *vault.VaultManager) error {
	nexuspool, err := cache.SearchAliveNexusPool()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No running nexuspool",
		})
	}

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

	session_id, err := cache.UpsertArenaSession(services.UUIDToString(user_id))

	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot create arena session",
		})
	}

	slog.Debug("arena session created", "session_id", session_id)

	ws_url := nexuspool.Url
	if strings.HasPrefix(ws_url, "http://") {
		ws_url = "ws://" + strings.TrimPrefix(ws_url, "http://")
	} else if strings.HasPrefix(ws_url, "https://") {
		ws_url = "wss://" + strings.TrimPrefix(ws_url, "https://")
	}

	sse_url := nexuspool.Url

	response.WS_Url = fmt.Sprintf("%s/ws/arena/", ws_url)
	response.SSE_Url = fmt.Sprintf("%s/sse/arena/", sse_url)

	var session_context struct {
		UserId    string `json:"user_id"`
		SessionId string `json:"session_id"`
	}
	session_context.SessionId = session_id
	session_context.UserId = services.UUIDToString(user_id)

	session_context_json, err := json.Marshal(session_context)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot create arena session",
		})
	}

	aes_key := vault.OpenNexusAESKey

	encrypted_session_context, err := security.EncryptAESUrlSafe(string(session_context_json), aes_key)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to encrypted compilation message",
		})
	}
	response.EncryptedSessionContext = encrypted_session_context

	err = cache.RefreshActiveModule(user_id, db, false)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(response)
}
