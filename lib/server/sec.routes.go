package server

import (
	m "backend/lib/maintenance"
	"backend/lib/server/middleware"
	"backend/lib/server/routes/security"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterSecurityRoutes() {

	security_group := server.App.Group("/security")

	server.registerSecurityEngineRoutes(security_group)
	server.registerSecurityApiRoutes(security_group)
	server.registerSecurityServicesRoutes(security_group)
}

func (server *MaintenanceServer) registerSecurityEngineRoutes(routes fiber.Router) {
	engines_group := routes.Group("/nexuspool")

	engines_group.Post("/init",
		middleware.OnMode(m.MODE_INIT),
		middleware.OnState(m.STATE_CONFIGURING),
		middleware.OnSubstate(m.SUBSTATE_CONFIGURING_SECURITY),
		middleware.WithKey("NEXUSPOOL_INIT_KEY", func() (string, error) {
			return server.VaultManager.GetApiKey("NEXUSPOOL_INIT_KEY")
		}),
		func(c *fiber.Ctx) error {
			return security.InitNexusPoolSecurityHandler(c, &server.VaultManager, func(result bool) {
				server.SecurityManager.ChanEnginesTokenApplication <- result
			})
		})

	engines_group.Get("/new", middleware.OnMode(m.MODE_OPERATIONAL),
		middleware.OnState(m.STATE_RUNNING),
		middleware.OnSubstate(m.SUBSTATE_SAFE),
		middleware.WithKey("NEXUSPOOL_ADM_KEY", func() (string, error) {
			return server.VaultManager.GetApiKey("NEXUSPOOL_ADM_KEY")
		}),
		func(c *fiber.Ctx) error {
			return security.RequestEngineConnexionHandler(c, &server.Cache, &server.VaultManager)
		},
	)

	engines_group.Post("/connect", middleware.OnMode(m.MODE_OPERATIONAL),
		middleware.OnState(m.STATE_RUNNING),
		middleware.OnSubstate(m.SUBSTATE_SAFE),
		middleware.WithKey("NEXUSPOOL_ADM_KEY", func() (string, error) {
			return server.VaultManager.GetApiKey("NEXUSPOOL_ADM_KEY")
		}),
		func(c *fiber.Ctx) error {
			return security.ConnectHandler(c, &server.Cache, &server.VaultManager)
		},
	)
}

func (server *MaintenanceServer) registerSecurityApiRoutes(routes fiber.Router) {
	api_group := routes.Group("/api")

	api_group.Post("/init",
		middleware.OnMode(m.MODE_INIT),
		middleware.OnState(m.STATE_CONFIGURING),
		middleware.OnSubstate(m.SUBSTATE_CONFIGURING_INIT),
		middleware.WithKey("API_INIT_KEY", nil),
		func(c *fiber.Ctx) error {
			return security.InitApiSecurityHandler(c, server.VaultManager.Api, func(result bool) {
				server.SecurityManager.ChanApiTokenApplication <- result
			})
		})
}

func (server *MaintenanceServer) registerSecurityServicesRoutes(routes fiber.Router) {
	services_group := routes.Group("/services")

	services_group.Post("/init",
		middleware.OnMode(m.MODE_INIT),
		middleware.OnState(m.STATE_CONFIGURING),
		middleware.OnSubstate(m.SUBSTATE_CONFIGURING_INIT),
		middleware.WithKey("SERVICES_INIT_KEY", func() (string, error) {
			return server.VaultManager.GetApiKey("SERVICES_INIT_KEY")
		}),
		func(c *fiber.Ctx) error {
			return security.InitServicesSecurityHandler(c, server.VaultManager.Services, func(result bool) {
				server.SecurityManager.ChanServicesTokenApplication <- result
			})
		})
}
