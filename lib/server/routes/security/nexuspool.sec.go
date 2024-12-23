package security

import (
	"backend/lib/services"
	v "backend/lib/vault"
	"log/slog"

	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"
)

func InitNexusPoolSecurityHandler(ctx *fiber.Ctx, manager *v.VaultManager, on_complete func(bool)) error {
	var data struct {
		VaultNexusPoolToken string `json:"vault_nexuspool_token"`
	}
	if err := ctx.BodyParser(&data); err != nil {
		on_complete(false)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if data.VaultNexusPoolToken == "" {
		on_complete(false)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "vault_nexuspool_token is required",
		})
	}

	manager.NexusPool.SetToken(data.VaultNexusPoolToken)

	on_complete(true)
	slog.Info("Successfully load vault nexuspools token")

	return ctx.JSON(fiber.Map{
		"message": "vault nexuspools token set successfully",
	})
}

func RequestNexusPoolConnexionHandler(ctx *fiber.Ctx, cache *services.Cache, manager *v.VaultManager) error {
	id := uuid.New().String()

	nexuspool := services.NexusPool{
		Id:    id,
		Alive: false,
	}

	if err := cache.AddNexusPool(nexuspool); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	aes_key, err := genAESKey()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := manager.StoreNexusPoolAESKey(id, aes_key); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	hmac_key, err := genHMACKey()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := manager.StoreNexusPoolHMACKey(id, hmac_key); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	slog.Info("NexusPool successfully created", "nexuspool", nexuspool)
	return ctx.JSON(nexuspool)
}

func ConnectHandler(ctx *fiber.Ctx, cache *services.Cache, manager *v.VaultManager) error {

	var data struct {
		Id       string `json:"id"`
		Password string `json:"password"`
		Url      string `json:"url"`
	}

	if err := ctx.BodyParser(&data); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	nexuspool, err := cache.GetNexusPool(data.Id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	key, err := manager.GetNexusPoolAESKey(nexuspool.Id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	password, err := DecryptAES(data.Password, key)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if password != "Hello World" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid password",
		})
	}

	nexuspool.Alive = true
	nexuspool.Url = data.Url
	cache.UpdateNexusPool(nexuspool)
	slog.Info("NexusPool successfully connected", "nexuspool", nexuspool)

	return ctx.JSON(fiber.Map{
		"status": "accepted",
	})
}
