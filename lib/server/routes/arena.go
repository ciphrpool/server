package routes

import (
	"backend/lib/database"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func ArenaUnregisteredHandler(ctx *fiber.Ctx, cache *database.Cache) error {
	engine, err := cache.SearchAliveEngine()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No running engine",
		})
	}

	var response struct {
		WS_Url    string `json:"ws_url"`
		SSE_Url   string `json:"sse_url"`
		SessionId string `json:"session_id"`
	}

	client_ip := ctx.IP()
	session_id, err := cache.UpsertArenaSession(client_ip)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot create arena session",
		})
	}

	ws_url := engine.Url
	if strings.HasPrefix(ws_url, "http://") {
		ws_url = "ws://" + strings.TrimPrefix(ws_url, "http://")
	} else if strings.HasPrefix(ws_url, "https://") {
		ws_url = "wss://" + strings.TrimPrefix(ws_url, "https://")
	}

	sse_url := engine.Url

	response.WS_Url = fmt.Sprintf("%s/ws/arena/", ws_url)
	response.SSE_Url = fmt.Sprintf("%s/sse/arena/", sse_url)
	response.SessionId = session_id

	return ctx.JSON(response)
}
