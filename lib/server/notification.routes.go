package server

import (
	"backend/lib/notifications"
	"backend/lib/server/middleware"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterNotificationRoutes() {
	notification_group := server.App.Group("/notify")
	notification_group.Use(middleware.Protected(&server.AuthService))
	notification_group.Use(middleware.RequireSession(&server.AuthService, server.Sessions))
	notification_group.Get("/refresh",
		func(c *fiber.Ctx) error {
			userID, err := middleware.GetUserID(c)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "unknown user",
				})
			}

			err = server.Notifications.RefreshConnectionTTL(c.Context(), userID, &server.Cache)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to refresh connection",
				})
			}

			return c.SendStatus(fiber.StatusOK)
		},
	)

	notification_group.Post("/close",
		func(c *fiber.Ctx) error {
			userID, err := middleware.GetUserID(c)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "unknown user",
				})
			}

			err = server.Notifications.CloseConnection(c.Context(), userID, &server.Cache)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to close connection",
				})
			}

			return c.SendStatus(fiber.StatusOK)
		},
	)

	notification_group.Get("/session",
		func(c *fiber.Ctx) error {
			return server.Notifications.SSENotificationHandler(c, &server.Cache)
		},
	)

	notification_group.Get("/ping",
		func(c *fiber.Ctx) error {
			userID, err := middleware.GetUserID(c)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "unknown user",
				})
			}

			server.Notifications.Send(
				c.Context(),
				notifications.TypePing,
				notifications.PriorityLow,
				userID,
				fiber.Map{
					"ping": "Hello World",
				},
				fiber.Map{},
			)

			return c.SendStatus(fiber.StatusAccepted)
		},
	)
}
