package routes

import (
	"backend/lib/server"

	"github.com/gofiber/fiber/v2"
)

func RegisterArenaApi(server *server.MaintenanceServer) {
	arena_group := server.App.Group("/arena")
	arena_group.Get("/unregistered", ArenaUnregisteredHandler)
}

func ArenaUnregisteredHandler(ctx *fiber.Ctx) error {

	return ctx.JSON("")
}
