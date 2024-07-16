package server

import (
	"backend/lib/database"
	"backend/lib/maintenance"
	"backend/lib/vault"

	"github.com/gofiber/fiber/v2"
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

func (server *MaintenanceServer) Start() {
	server.SecurityManager.Start(&server.StateMachine)

	server.StateMachine.When(
		maintenance.MODE_INIT,
		maintenance.STATE_CONFIGURING,
		maintenance.SUBSTATE_CONFIGURING_SERVICES,
		func() {
			// Connect all services
			cache_pwd, err := server.VaultManager.GetCachePwd()
			if err != nil {
				// raise fault
				return
			}
			err = server.Cache.Connect(cache_pwd)
			if err != nil {
				// raise fault
				return
			}
			server.StateMachine.To(maintenance.MODE_INIT, maintenance.STATE_CONFIGURING, maintenance.SUBSTATE_CONFIGURING_SECURITY)
		})

}
