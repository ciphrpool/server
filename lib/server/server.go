package server

import (
	"backend/lib/authentication"
	"backend/lib/maintenance"
	"backend/lib/notifications"
	"backend/lib/server/middleware"
	"backend/lib/services"
	"backend/lib/vault"
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type MaintenanceServer struct {
	*fiber.App
	Db              services.Database
	Cache           services.Cache
	Notifications   *notifications.NotificationService
	Sessions        *session.Store
	VaultManager    vault.VaultManager
	SecurityManager maintenance.SecurityManager
	StateMachine    maintenance.StateMachine
	AuthService     *authentication.AuthService
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
	notificationConfig := &notifications.Config{
		WorkerCount:       10,               // Number of workers
		WorkerQueueSize:   1000,             // Size of job queue
		RetryAttempts:     3,                // Number of retry attempts
		RetryDelay:        time.Second * 2,  // Delay between retries
		ShutdownTimeout:   time.Second * 30, // Timeout for graceful shutdown
		HeartbeatInterval: time.Second * 10, // Health check interval
		InitialPoolSize:   100,              // Initial size of object pool
		SessionTimeout:    time.Hour * 24,   // Session duration
	}
	cache := services.DefaultCache()
	notifications, err := notifications.NewNotificationService(notificationConfig)
	if err != nil {
		return nil, err
	}

	server := MaintenanceServer{
		App:             fiber.New(),
		Db:              services.DefaultDatabase(),
		Cache:           cache,
		Notifications:   notifications,
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

	// Initialize session store
	store := session.New(session.Config{
		Expiration:     24 * time.Hour,
		KeyLookup:      "cookie:session", // "<source>:<key>"
		CookiePath:     "/",
		CookieSecure:   false, // Set to true in production with HTTPS
		CookieHTTPOnly: true,
	})
	server.Sessions = store

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
			db_pwd, err := server.VaultManager.GetDbPwd()
			if err != nil {
				slog.Error("Db pwd retrieval failed", "error", err)
				// raise fault
				return
			}
			err = server.Cache.Connect(cache_pwd)
			if err != nil {
				// raise fault
				slog.Error("Cache connection failed", "error", err)
				return
			}
			err = server.Db.Connect(db_pwd)
			if err != nil {
				// raise fault
				slog.Error("Db connection failed", "error", err)
				return
			}

			auth_config, err := authentication.BuildAuthConfig(&server.VaultManager)
			if err != nil {
				// raise fault
				slog.Error("Cannot build Auth Service Config", "error", err)
				return
			}
			auth_service, err := authentication.NewAuthService(auth_config)
			if err != nil {
				// raise fault
				slog.Error("Cannot build Auth Service", "error", err)
				return
			}
			server.AuthService = auth_service

			if err := server.Notifications.Start(context.Background(), &server.Cache); err != nil {
				// raise fault
				slog.Error("Notifications could not start", "error", err)
				return
			}

			server.StateMachine.To(maintenance.MODE_INIT, maintenance.STATE_CONFIGURING, maintenance.SUBSTATE_CONFIGURING_SECURITY)
		})

}
