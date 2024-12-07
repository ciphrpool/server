package routes

import (
	"backend/lib/notifications"
	"backend/lib/server/middleware"
	"backend/lib/services"
	"context"
	"fmt"
	"log/slog"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/gofiber/fiber/v2"
)

type FriendRequestData struct {
	FriendTag string `json:"friend_tag"`
}

func FriendRequestHandler(data FriendRequestData, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)
	friend, err := queries.GetUserIDByTag(query_ctx, data.FriendTag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	err = queries.CreateFriendRequest(query_ctx, basepool.CreateFriendRequestParams{
		User1ID: user_id,
		User2ID: friend.ID,
	})

	if err != nil {
		slog.Debug("CreateUserRelationship failed", "error", err)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot create this relationship",
		})
	}

	return ctx.SendStatus(fiber.StatusAccepted)
}

type RemoveFriendData struct {
	FriendTag string `json:"friend_tag"`
}

func RemoveFriendHandler(data RemoveFriendData, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)
	friend, err := queries.GetUserIDByTag(query_ctx, data.FriendTag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	_, err = queries.RemoveFriendship(query_ctx, basepool.RemoveFriendshipParams{
		User1ID: user_id,
		User2ID: friend.ID,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	return ctx.SendStatus(fiber.StatusAccepted)
}

type RemovePendingFriendRequestData struct {
	FriendTag string `json:"friend_tag"`
}

func RemovePendingFriendRequestHandler(data RemovePendingFriendRequestData, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)
	friend, err := queries.GetUserIDByTag(query_ctx, data.FriendTag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	_, err = queries.RemovePendingRequest(query_ctx, basepool.RemovePendingRequestParams{
		User1ID: user_id,
		User2ID: friend.ID,
	})

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	return ctx.SendStatus(fiber.StatusAccepted)
}

type FriendResponseData struct {
	RequesterTag string `json:"requester_tag"`
	Response     bool   `json:"response"`
}

func FriendResponceHandler(data FriendResponseData, ctx *fiber.Ctx, db *services.Database, notify *notifications.NotificationService) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)
	requester, err := queries.GetUserIDByTag(query_ctx, data.RequesterTag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot find this user",
		})
	}

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	if data.Response {
		_, err = queries.AcceptFriendRequest(query_ctx, basepool.AcceptFriendRequestParams{
			User1ID: requester.ID,
			User2ID: user_id,
		})
		notify.Send(
			ctx.Context(),
			notifications.TypeMessage,
			notifications.PriorityMedium,
			requester.ID,
			fiber.Map{
				"type": "relationship",
				"msg":  fmt.Sprintf("%s#%s has accepted your friend request !", requester.Username, requester.Tag),
			},
			fiber.Map{},
		)
	} else {
		_, err = queries.RejectFriendRequest(query_ctx, basepool.RejectFriendRequestParams{
			User1ID: requester.ID,
			User2ID: user_id,
		})
	}

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot update this relationship status",
		})
	}

	return ctx.SendStatus(fiber.StatusAccepted)
}

func GetAllFriendsHandler(ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	user_id, err := middleware.GetUserID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	friends, err := queries.GetAllFriends(query_ctx, user_id)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"friends": friends,
	})
}
