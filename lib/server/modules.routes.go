package server

import (
	m "backend/lib/maintenance"
	"backend/lib/server/middleware"
	"backend/lib/server/routes"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterModulesRoutes() {
	modules_group := server.App.Group("/modules")
	modules_group.Use(
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
	)

	modules_group.Use(middleware.Protected(&server.AuthService))

	modules_group.Get("/summary/all",
		func(c *fiber.Ctx) error {
			return routes.GetAllModulesHandler(c, &server.Db)
		},
	)
	modules_group.Post("/create",
		middleware.RequireSession(&server.AuthService, server.Sessions),
		func(c *fiber.Ctx) error {
			var data routes.ActivateModuleData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.ActivateModuleHandler(data, c, &server.Db)
		},
	)
	modules_group.Post("/activate",
		middleware.RequireSession(&server.AuthService, server.Sessions),
		func(c *fiber.Ctx) error {
			var data routes.CreateModuleData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.CreateModuleHandler(data, c, &server.Db)
		},
	)
	modules_group.Get("/fetch",
		func(c *fiber.Ctx) error {
			var params routes.FetchModuleParams
			if err := c.QueryParser(&params); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid query parameters",
				})
			}
			return routes.FetchModuleHandler(params, c, &server.Db)
		},
	)
	modules_group.Post("/rename",
		middleware.RequireSession(&server.AuthService, server.Sessions),
		func(c *fiber.Ctx) error {
			var data routes.RenameModuleData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.RenameModuleHandler(data, c, &server.Db)
		},
	)

	modules_group.Post("/push",
		middleware.RequireSession(&server.AuthService, server.Sessions),
		func(c *fiber.Ctx) error {
			var data routes.PushModuleData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.PushModuleHandler(data, c, &server.Db, &server.VaultManager)
		},
	)

	modules_group.Post("/delete",
		middleware.RequireSession(&server.AuthService, server.Sessions),
		func(c *fiber.Ctx) error {
			var data routes.DeleteModuleData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.DeleteModuleHandler(data, c, &server.Db)
		},
	)

	modules_group.Get("/prepare_compilation",
		middleware.RequireSession(&server.AuthService, server.Sessions),
		func(c *fiber.Ctx) error {
			return routes.PrepareCompilationHandler(c, &server.VaultManager)
		},
	)
}
