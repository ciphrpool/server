package routes

import (
	"backend/lib/server/middleware"
	"backend/lib/services"
	"context"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/gofiber/fiber/v2"
)

type SearchByUsernameParams struct {
	Username string `query:"username"`
	Detailed bool   `query:"detailed"`
}

func SearchByUsernameHandler(params SearchByUsernameParams, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	if params.Detailed {
		users, err := queries.GetUserByUsernameDetailed(query_ctx, params.Username)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "cannot search users matching the given username",
			})
		}
		return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"users": users,
		})
	} else {
		users, err := queries.GetUserByUsernameSummary(query_ctx, params.Username)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "cannot search users matching the given username",
			})
		}
		return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"users": users,
		})
	}
}

type GetUserByTagParams struct {
	Tag      string `query:"tag"`
	Detailed bool   `query:"detailed"`
}

func GetUserByTagHandler(params GetUserByTagParams, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	if params.Detailed {
		user, err := queries.GetUserByTagDetailed(query_ctx, params.Tag)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "user not found",
			})
		}
		return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"user": user,
		})
	} else {
		user, err := queries.GetUserByTagSummary(query_ctx, params.Tag)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "user not found",
			})
		}
		return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"user": user,
		})
	}
}

type GetSelfParams struct {
}

func GetSelfHandler(params GetSelfParams, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}
	user, err := queries.GetUserByID(query_ctx, user_id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "user not found",
		})
	}
	return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"user": user,
	})
}
