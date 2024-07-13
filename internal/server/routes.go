package server

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func (server *MaintenanceServer) RegisterRoutes() {
	server.App.Get("/", server.HelloWorldHandler)
	server.App.Get("/health", server.healthHandler)

}

func (server *MaintenanceServer) HelloWorldHandler(c *fiber.Ctx) error {
	resp := map[string]string{
		"message": "Hello World",
	}
	return c.JSON(resp)
}

func (server *MaintenanceServer) healthHandler(c *fiber.Ctx) error {
	resp := map[string]string{
		"db": strconv.FormatBool(server.db.Health()),
	}
	return c.JSON(resp)
}
