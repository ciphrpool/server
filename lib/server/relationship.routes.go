package server

import (
	m "backend/lib/maintenance"
	"backend/lib/server/middleware"
	"backend/lib/server/routes"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterRelationshipRoutes() {
	relationship_group := server.App.Group("/relationship")

	request_group := relationship_group.Group("/request")

	request_group.Use(middleware.ForAuthentificatedUser(func() (string, error) {
		return server.VaultManager.GetApiKey("MCS_JWT_KEY")
	}))

	request_group.Post("/friend",
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
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
	request_group.Post("/block",
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
		func(c *fiber.Ctx) error {
			var data routes.BlockUserData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}

			return routes.BlockUserHandler(data, c, &server.Db)
		},
	)
	request_group.Post("/unblock",
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
		func(c *fiber.Ctx) error {
			var data routes.UnblockUserData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}

			return routes.UnblockUserHandler(data, c, &server.Db)
		},
	)
	request_group.Post("/unfriend",
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
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
	request_group.Post("/clear_pending",
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
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
	request_group.Post("/accept",
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
		func(c *fiber.Ctx) error {
			var data routes.FriendResponseData

			if err := c.BodyParser(&data); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}

			return routes.FriendResponceHandler(data, c, &server.Db)
		},
	)
	request_group.Get("/all_friends",
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
		func(c *fiber.Ctx) error {
			return routes.GetAllFriendsHandler(c, &server.Db)
		},
	)
}
