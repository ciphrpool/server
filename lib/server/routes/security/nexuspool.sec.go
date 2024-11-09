package security

import (
	"backend/lib"
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
	slog.Info("Successfully load vault engines token")

	return ctx.JSON(fiber.Map{
		"message": "vault engines token set successfully",
	})
}

func RequestEngineConnexionHandler(ctx *fiber.Ctx, cache *services.Cache, manager *v.VaultManager) error {
	id := uuid.New().String()

	nexuspool := lib.NexusPool{
		Id:    id,
		Alive: false,
	}

	if err := cache.AddEngine(nexuspool); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	key, err := genAESKey()
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := manager.StoreNexusPoolAESKey(id, key); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	slog.Info("Engine successfully created", "engine", nexuspool)
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
	engine, err := cache.GetEngine(data.Id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	key, err := manager.GetNexusPoolAESKey(engine.Id)
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

	engine.Alive = true
	engine.Url = data.Url
	cache.UpdateEngine(engine)
	slog.Info("Engine successfully connected", "engine", engine)

	return ctx.JSON(fiber.Map{
		"status": "accepted",
	})
}
