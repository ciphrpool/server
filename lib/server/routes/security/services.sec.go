package security

import (
	v "backend/lib/vault"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

func InitServicesSecurityHandler(ctx *fiber.Ctx, vault *v.Vault, on_complete func(bool)) error {
	var data struct {
		VaultServicesToken string `json:"vault_services_token"`
	}
	if err := ctx.BodyParser(&data); err != nil {
		on_complete(false)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if data.VaultServicesToken == "" {
		on_complete(false)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "vault_services_token is required",
		})
	}

	vault.SetToken(data.VaultServicesToken)
	on_complete(true)
	slog.Info("Successfully load vault services token")

	return ctx.JSON(fiber.Map{
		"message": "vault services token set successfully",
	})
}
