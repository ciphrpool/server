package server

import (
	"backend/internal/database"

	"github.com/gofiber/fiber/v2"
)

type MaintenanceServer struct {
	*fiber.App
	db database.Service
}

func New() *MaintenanceServer {
	server := &MaintenanceServer{
		App: fiber.New(),
		db:  database.New(),
	}

	return server
}
