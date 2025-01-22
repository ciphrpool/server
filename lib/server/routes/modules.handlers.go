package routes

import (
	"backend/lib/server/middleware"
	"backend/lib/server/routes/security"
	"backend/lib/services"
	"backend/lib/vault"
	"context"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/gofiber/fiber/v2"
)

func GetAllModulesHandler(ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	modules, err := queries.GetAllModulesSummaryByUserID(query_ctx, user_id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"modules": modules,
	})
}

type FetchModuleParams struct {
	Name string `query:"name"`
}

func FetchModuleHandler(params FetchModuleParams, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	module, err := queries.FetchModule(query_ctx, basepool.FetchModuleParams{
		UserID: user_id,
		Name:   params.Name,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "invalid module name",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"module": module,
	})
}

type RenameModuleData struct {
	NewName  string `json:"new_name"`
	PrevName string `json:"prev_name"`
}

func RenameModuleHandler(data RenameModuleData, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	err = queries.RenameModule(query_ctx, basepool.RenameModuleParams{
		UserID:   user_id,
		NewName:  data.NewName,
		PrevName: data.PrevName,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "invalid module name",
		})
	}

	return ctx.SendStatus(fiber.StatusOK)
}

type PushModuleData struct {
	Name string `json:"name"`
	Code string `json:"code"`
	Hmac string `json:"hmac"`
}

func PushModuleHandler(data PushModuleData, ctx *fiber.Ctx, cache *services.Cache, db *services.Database, vault *vault.VaultManager) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	hmac_key := vault.OpenNexusHMACKey
	res, err := security.CheckHMACwithUserID([]byte(hmac_key), services.UUIDToString(user_id), data.Code, data.Hmac)
	if err != nil || !res {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "invalid module, please compile the module before pushing it",
		})
	}

	err = queries.PushModule(query_ctx, basepool.PushModuleParams{
		UserID: user_id,
		Name:   data.Name,
		Code:   data.Code,
		Hmac:   data.Hmac,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "invalid module name",
		})
	}

	err = cache.RefreshActiveModule(user_id, db, true)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.SendStatus(fiber.StatusOK)
}

type DeleteModuleData struct {
	Name string `json:"name"`
}

func DeleteModuleHandler(data DeleteModuleData, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	err = queries.DeleteModule(query_ctx, basepool.DeleteModuleParams{
		UserID: user_id,
		Name:   data.Name,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "invalid module name",
		})
	}

	return ctx.SendStatus(fiber.StatusOK)
}

type CreateModuleData struct {
	Name string `json:"name"`
}

func CreateModuleHandler(data CreateModuleData, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	_, err = queries.CreateModule(query_ctx, basepool.CreateModuleParams{
		UserID: user_id,
		Name:   data.Name,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "invalid module name",
		})
	}

	return ctx.SendStatus(fiber.StatusOK)
}

type ActivateModuleData struct {
	Name string `json:"name"`
}

func ActivateModuleHandler(data ActivateModuleData, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	err = queries.ActivateModule(query_ctx, basepool.ActivateModuleParams{
		UserID: user_id,
		Name:   data.Name,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "invalid module name",
		})
	}

	return ctx.SendStatus(fiber.StatusOK)
}

func PrepareCompilationHandler(ctx *fiber.Ctx, vault *vault.VaultManager) error {

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	aes_key := vault.OpenNexusAESKey
	encrypted_user_id, err := security.EncryptAES(services.UUIDToString(user_id), aes_key)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to encrypted compilation message",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"encrypted_user_id": encrypted_user_id,
	})
}
