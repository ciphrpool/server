package routes

import (
	"backend/lib/services"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func ArenaUnregisteredHandler(ctx *fiber.Ctx, cache *services.Cache) error {
	nexuspool, err := cache.SearchAliveNexusPool()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No running nexuspool",
		})
	}

	var response struct {
		WS_Url    string `json:"ws_url"`
		SSE_Url   string `json:"sse_url"`
		SessionId string `json:"session_id"`
	}

	client_ip := ctx.IP()
	session_id, err := cache.UpsertArenaUnregisteredSession(client_ip)
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

	response.WS_Url = fmt.Sprintf("%s/ws/arena/unregistered/", ws_url)
	response.SSE_Url = fmt.Sprintf("%s/sse/arena/", sse_url)
	response.SessionId = session_id

	return ctx.JSON(response)
}
