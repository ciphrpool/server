package server

import (
	"backend/lib/database"
	"backend/lib/maintenance"
	"backend/lib/server/middleware"
	"backend/lib/vault"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type MaintenanceServer struct {
	*fiber.App
	Db              database.Database
	Cache           database.Cache
	VaultManager    vault.VaultManager
	SecurityManager maintenance.SecurityManager
	StateMachine    maintenance.StateMachine
}

func New() (*MaintenanceServer, error) {
	vault_manager, err := vault.NewVaultManager()
	if err != nil {
		return nil, err
	}
	security_manager, err := maintenance.NewSecurityManager()
	if err != nil {
		return nil, err
	}

	cache, err := database.NewCache()
	if err != nil {
		return nil, err
	}
	server := MaintenanceServer{
		App:             fiber.New(),
		Db:              database.New(),
		Cache:           cache,
		VaultManager:    vault_manager,
		SecurityManager: security_manager,
		StateMachine:    maintenance.NewStateMachine(),
	}

	return &server, nil
}

func (server *MaintenanceServer) Configure() {
	err := maintenance.InitLogger("mcs.log")
	if err == nil {
		server.App.Use(middleware.Logger())
	}
	server.App.Use(func(c *fiber.Ctx) error {
		c.Locals("StateMachine", &server.StateMachine)
		return c.Next()
	})

	server.App.Use(helmet.New())
	server.App.Use(recover.New())
	server.App.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
}

func (server *MaintenanceServer) Start() {
	slog.Info("Starting the server")

	server.Configure()
	server.RegisterRoutes()
	server.SecurityManager.Start(&server.StateMachine)

	server.StateMachine.When(
		maintenance.MODE_INIT,
		maintenance.STATE_CONFIGURING,
		maintenance.SUBSTATE_CONFIGURING_SERVICES,
		func() {
			slog.Info("Connecting services ...")
			// Connect all services
			cache_pwd, err := server.VaultManager.GetCachePwd()
			if err != nil {
				slog.Error("Cache pwd retrieval failed", "error", err)
				// raise fault
				return
			}
			err = server.Cache.Connect(cache_pwd)
			if err != nil {
				// raise fault
				slog.Error("Cache connection failed", "error", err)
				return
			}
			server.StateMachine.To(maintenance.MODE_INIT, maintenance.STATE_CONFIGURING, maintenance.SUBSTATE_CONFIGURING_SECURITY)
		})

}
