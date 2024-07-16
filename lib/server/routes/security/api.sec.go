package security

import (
	v "backend/lib/vault"
	"os"

	"github.com/gofiber/fiber/v2"
)

func InitApiSecurityHandler(ctx *fiber.Ctx, vault *v.Vault, on_complete func(bool)) error {
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

	vault.SetToken(data.VaultApiToken)
	on_complete(true)

	os.Unsetenv("API_INIT_KEY")
	return ctx.JSON(fiber.Map{
		"message": "vault api token set successfully",
	})
}
