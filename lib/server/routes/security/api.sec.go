package security

import (
	v "backend/lib/vault"
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
)

func InitApiSecurityHandler(ctx *fiber.Ctx, manager *v.VaultManager, on_complete func(bool)) error {
	var data struct {
		VaultApiToken string `json:"vault_api_token"`
	}
	if err := ctx.BodyParser(&data); err != nil {
		on_complete(false)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if data.VaultApiToken == "" {
		on_complete(false)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "vault_api_token is required",
		})
	}

	manager.Api.SetToken(data.VaultApiToken)

	if err := manager.LoadApiKeys("SERVICES_INIT_KEY", "NEXUSPOOL_INIT_KEY", "NEXUSPOOL_ADM_KEY", "JWT_KEY"); err != nil {
		slog.Error("api could not be loaded", "error", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "api could not be loaded",
		})
	}

	on_complete(true)
	slog.Info("Successfully load vault api token")

	os.Unsetenv("API_INIT_KEY")
	return ctx.JSON(fiber.Map{
		"message": "vault api token set successfully",
	})
}
