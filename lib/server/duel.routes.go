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
	duel_group.Use(middleware.Protected(&server.AuthService))
	duel_group.Use(middleware.RequireSession(&server.AuthService, server.Sessions))

	friendlies_group := duel_group.Group("/friendly")

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

	friendlies_group.Get("/prepare",
		func(c *fiber.Ctx) error {
			var params routes.FriendliesPrepareResponseParams

			if err := c.QueryParser(&params); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.FriendliesPrepareHandler(params, c, &server.Cache, &server.Db, &server.VaultManager, server.Notifications)
		},
	)
	duel_group.Get("/session",
		func(c *fiber.Ctx) error {
			var params routes.GetDuelSessionDataParams

			if err := c.QueryParser(&params); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.GetDuelSessionDataHandler(params, c, &server.Cache, &server.Db, &server.VaultManager, server.Notifications)
		},
	)
	duel_group.Get("/result",
		func(c *fiber.Ctx) error {
			var params routes.GetDuelResultParams

			if err := c.QueryParser(&params); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.GetDuelResultHandler(params, c, &server.Cache, &server.Db, &server.VaultManager, server.Notifications)
		},
	)

	duel_group.Get("/history",
		func(c *fiber.Ctx) error {
			var params routes.GetDuelHistoryParams

			if err := c.QueryParser(&params); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid request body",
				})
			}
			return routes.GetDuelHistoryHandler(params, c, &server.Cache, &server.Db, &server.VaultManager, server.Notifications)
		},
	)
	// test_duel_group := server.App.Group("/test_duel")

	// test_duel_group.Post("/result",
	// 	func(c *fiber.Ctx) error {
	// 		type TestDuelPlayerSummaryData struct {
	// 			PID      string `json:"pid"`
	// 			Elo      uint   `json:"elo"`
	// 			Tag      string `json:"tag"`
	// 			Username string `json:"username"`
	// 		}

	// 		type TestDDuelSessionData struct {
	// 			P1       TestDuelPlayerSummaryData `json:"p1"`
	// 			P2       TestDuelPlayerSummaryData `json:"p2"`
	// 			DuelType string                    `json:"duel_type"`
	// 		}

	// 		type TestDuelResult struct {
	// 			P1Summary   duels.Summary        `json:"p1_summary"`
	// 			P2Summary   duels.Summary        `json:"p2_summary"`
	// 			Outcome     duels.Outcome        `json:"outcome"`
	// 			SessionData TestDDuelSessionData `json:"session_data"`
	// 			SessionID   string               `json:"session_id"`
	// 		}
	// 		var data TestDuelResult

	// 		if err := c.BodyParser(&data); err != nil {
	// 			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 				"error": "invalid request body",
	// 			})
	// 		}
	// 		// Validate session ID
	// 		if data.SessionID == "" {
	// 			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 				"error": "session_id is required",
	// 			})
	// 		}

	// 		// Create the channel key
	// 		channel := fmt.Sprintf("duel:result:%s", data.SessionID)

	// 		// Convert the data to JSON
	// 		result_json, err := json.Marshal(data)
	// 		if err != nil {
	// 			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 				"error": "failed to marshal result data",
	// 			})
	// 		}

	// 		// Publish to Redis
	// 		if err := server.Cache.Db.Publish(c.Context(), channel, result_json).Err(); err != nil {
	// 			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
	// 				"error": "failed to publish result",
	// 			})
	// 		}

	// 		return c.Status(fiber.StatusOK).JSON(fiber.Map{
	// 			"message": "duel result published successfully",
	// 			"channel": channel,
	// 		})
	// 	},
	// )
}
