package server

import (
	m "backend/lib/maintenance"
	"backend/lib/server/middleware"
	"backend/lib/server/routes"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterDuelRoutes() {
	duel_group := server.App.Group("/duel")
	duel_group.Use(
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
	)
	friendlies_group := duel_group.Group("/friendlies")

	friendlies_group.Use(middleware.Protected(&server.AuthService))
	friendlies_group.Use(middleware.RequireSession(&server.AuthService, server.Sessions))

	friendlies_group.Post("/challenge",
		func(c *fiber.Ctx) error {
			var data routes.FriendliesChallengeData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.FriendliesChallengeHandler(data, c, &server.Cache, &server.Db, server.Notifications)
		},
	)

	friendlies_group.Post("/response",
		func(c *fiber.Ctx) error {
			var data routes.FriendliesChallengeResponseData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.FriendliesChallengeResponeHandler(data, c, &server.Cache, &server.Db, &server.VaultManager, server.Notifications)
		},
	)
}
