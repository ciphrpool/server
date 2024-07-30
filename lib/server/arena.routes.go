package server

import (
	m "backend/lib/maintenance"
	"backend/lib/server/middleware"
	"backend/lib/server/routes"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterArenaRoutes() {
	arena_group := server.App.Group("/arena")
	arena_group.Get("/unregistered",
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
		func(c *fiber.Ctx) error {
			return routes.ArenaUnregisteredHandler(c, &server.Cache)
		},
	)
}
