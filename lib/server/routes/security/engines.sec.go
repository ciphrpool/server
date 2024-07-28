package security

import (
	"backend/lib"
	"backend/lib/database"
	v "backend/lib/vault"
	"log/slog"

	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"
)

func InitEngineSecurityHandler(ctx *fiber.Ctx, manager *v.VaultManager, on_complete func(bool)) error {
	var data struct {
		VaultEnginesToken string `json:"vault_engines_token"`
	}
	if err := ctx.BodyParser(&data); err != nil {
		on_complete(false)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if data.VaultEnginesToken == "" {
		on_complete(false)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "vault_engines_token is required",
		})
	}

	manager.Engines.SetToken(data.VaultEnginesToken)

	on_complete(true)
	slog.Info("Successfully load vault engines token")

	return ctx.JSON(fiber.Map{
		"message": "vault engines token set successfully",
	})
}

func RequestEngineConnexionHandler(ctx *fiber.Ctx, cache *database.Cache, manager *v.VaultManager) error {
	id := uuid.New().String()

	engine := lib.Engine{
		Id:    id,
		Alive: false,
	}

	if err := cache.AddEngine(engine); err != nil {
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

	if err := manager.StoreEngineAESKey(id, key); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	slog.Info("Engine successfully created", "engine", engine)
	return ctx.JSON(engine)
}

func ConnectHandler(ctx *fiber.Ctx, cache *database.Cache, manager *v.VaultManager) error {

	var data struct {
		Id       string `json:"id"`
		Password string `json:"password"`
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

	key, err := manager.GetEngineAESKey(engine.Id)
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
	cache.UpdateEngine(engine)
	slog.Info("Engine successfully connected", "engine", engine)

	return ctx.JSON(fiber.Map{
		"status": "accepted",
	})
}
