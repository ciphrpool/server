package server

import (
	m "backend/lib/maintenance"
	"backend/lib/server/middleware"
	"backend/lib/server/routes"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterRelationshipRoutes() {
	relationship_group := server.App.Group("/relationship")
	relationship_group.Use(
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
	)

	relationship_group.Use(middleware.Protected(&server.AuthService))

	relationship_group.Post("/friend",
		func(c *fiber.Ctx) error {
			var data routes.FriendRequestData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}

			return routes.FriendRequestHandler(data, c, &server.Db)
		},
	)

	relationship_group.Post("/unfriend",
		func(c *fiber.Ctx) error {
			var data routes.RemoveFriendData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}

			return routes.RemoveFriendHandler(data, c, &server.Db)
		},
	)

	relationship_group.Post("/clear_pending",
		func(c *fiber.Ctx) error {
			var data routes.RemovePendingFriendRequestData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}

			return routes.RemovePendingFriendRequestHandler(data, c, &server.Db)
		},
	)
	relationship_group.Post("/response",
		func(c *fiber.Ctx) error {
			var data routes.FriendResponseData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}

			return routes.FriendResponceHandler(data, c, &server.Db, server.Notifications)
		},
	)
	relationship_group.Get("/all_friends",
		func(c *fiber.Ctx) error {
			return routes.GetAllFriendsHandler(c, &server.Db)
		},
	)
}
