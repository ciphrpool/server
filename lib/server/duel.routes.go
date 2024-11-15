package server

import (
	m "backend/lib/maintenance"
	"backend/lib/server/middleware"
	"backend/lib/server/routes"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterDuelRoutes() {
	duel_group := server.App.Group("/duel")
	friendlies_group := duel_group.Group("/friendlies")

	friendlies_group.Use(middleware.ForAuthentificatedUser(func() (string, error) {
		return server.VaultManager.GetApiKey("MCS_JWT_KEY")
	}))

	friendlies_group.Post("/challenge",
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
		func(c *fiber.Ctx) error {
			var data routes.FriendliesChallengeData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.FriendliesChallengeHandler(data, c, &server.Cache, &server.Db)
		},
	)
}
