package server

import (
	m "backend/lib/maintenance"
	"backend/lib/server/middleware"
	"backend/lib/server/routes"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterUserRoutes() {
	users_group := server.App.Group("/users")
	users_group.Use(
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
	)
	public_group := users_group.Group("/public")
	private_group := users_group.Group("/private")

	private_group.Use(middleware.Protected(&server.AuthService))

	public_group.Get("/search",
		func(c *fiber.Ctx) error {
			var params routes.SearchByUsernameParams
			if err := c.QueryParser(&params); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid query parameters",
				})
			}

			return routes.SearchByUsernameHandler(params, c, &server.Db)
		},
	)
	public_group.Get("/tag",
		func(c *fiber.Ctx) error {
			var params routes.GetUserByTagParams
			if err := c.QueryParser(&params); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid query parameters",
				})
			}
			return routes.GetUserByTagHandler(params, c, &server.Db)
		},
	)

	private_group.Get("/me",
		func(c *fiber.Ctx) error {
			var params routes.GetSelfParams
			if err := c.QueryParser(&params); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid query parameters",
				})
			}
			return routes.GetSelfHandler(params, c, &server.Db)
		},
	)
}
