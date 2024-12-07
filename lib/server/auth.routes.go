package server

import (
	m "backend/lib/maintenance"
	"backend/lib/server/middleware"
	"backend/lib/server/routes"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterAuthRoutes() {
	auth_group := server.App.Group("/auth")
	auth_group.Use(
		middleware.OnMSS(m.MODE_OPERATIONAL, m.STATE_RUNNING, m.SUBSTATE_SAFE),
	)

	// OAuth routes
	auth_group.Get("/login/:provider", func(c *fiber.Ctx) error {
		provider := basepool.AuthType(c.Params("provider"))

		return routes.InitiateOAuthHandler(provider, c, server.AuthService, &server.Cache)
	})
	auth_group.Get("/callback/:provider", func(c *fiber.Ctx) error {
		provider := basepool.AuthType(c.Params("provider"))

		var params routes.OAuthCallbackParams
		if err := c.QueryParser(&params); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid query parameters",
			})
		}

		return routes.OAuthCallbackHandler(provider, params, c, server.AuthService, &server.Cache, &server.Db, &server.VaultManager, server.Sessions)
	})

	// Token management
	auth_group.Get("/refresh/session",
		func(c *fiber.Ctx) error {
			return routes.RefreshSessionHandler(c, server.AuthService, &server.Cache, server.Sessions)
		},
	)
	auth_group.Get("/refresh/access",
		middleware.RequireSession(&server.AuthService, server.Sessions),
		func(c *fiber.Ctx) error {
			return routes.RefreshAccessTokenHandler(c, server.AuthService, &server.Cache)
		},
	)
	auth_group.Get("/refresh/refresh",
		middleware.RequireSession(&server.AuthService, server.Sessions),
		func(c *fiber.Ctx) error {
			return routes.RefreshAllTokenHandler(c, server.AuthService, &server.Cache)
		},
	)

	auth_group.Post("/logout",
		middleware.Protected(&server.AuthService),
		func(c *fiber.Ctx) error {
			return routes.LogoutHandler(c, server.AuthService, &server.Cache, server.Sessions)
		},
	)
}
