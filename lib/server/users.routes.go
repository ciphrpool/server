package server

import (
	m "backend/lib/maintenance"
	"backend/lib/server/middleware"
	"backend/lib/server/routes"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterUserRoutes() {
	users_group := server.App.Group("/users")

	public_group := users_group.Group("/public")
	private_group := users_group.Group("/private")

	private_group.Use(middleware.ForAuthentificatedUser(func() (string, error) {
		return server.VaultManager.GetApiKey("MCS_JWT_KEY")
	}))

	public_group.Get("/search",
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
		func(c *fiber.Ctx) error {
			var params routes.SearchByUsernameParams
			if err := c.QueryParser(&params); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid query parameters",
				})
			}

			return routes.SearchByUsername(params, c, &server.Db)
		},
	)
	public_group.Get("/tag",
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
		func(c *fiber.Ctx) error {
			var params routes.GetUserByTagParams
			if err := c.QueryParser(&params); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid query parameters",
				})
			}
			return routes.GetUserByTag(params, c, &server.Db)
		},
	)
}
