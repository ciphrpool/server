package server

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterRoutes() {
	server.App.Get("/", server.HelloWorldHandler)
	server.App.Get("/health", server.healthHandler)

	server.RegisterSecurityRoutes()

}

func (server *MaintenanceServer) HelloWorldHandler(c *fiber.Ctx) error {
	resp := map[string]string{
		"message": "Hello World",
	}
	return c.JSON(resp)
}

func (server *MaintenanceServer) healthHandler(c *fiber.Ctx) error {
	resp := map[string]string{
		"db":    strconv.FormatBool(server.Db.Health()),
		"vault": strconv.FormatBool(server.VaultManager.Health()),
	}
	return c.JSON(resp)
}
