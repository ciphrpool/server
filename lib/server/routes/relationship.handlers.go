package routes

import (
	"backend/lib/server/middleware"
	"backend/lib/services"
	"context"
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
	friend_id, err := queries.GetUserIDByTag(query_ctx, data.FriendTag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	user_ctx, ok := ctx.Locals(middleware.USER_CONTEXT_KEY).(middleware.UserContext)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unknown user",
		})
	}
	err = queries.CreateUserRelationship(query_ctx, basepool.CreateUserRelationshipParams{
		User1ID:            user_ctx.UserID,
		User2ID:            friend_id,
		RelationshipType:   basepool.RelationshipFriend,
		RelationshipStatus: basepool.RelationshipStatusPending,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot create this relationship",
		})
	}

	return ctx.SendStatus(fiber.StatusAccepted)
}

type BlockUserData struct {
	BlockedTag string `json:"blocked_tag"`
}

func BlockUserHandler(data BlockUserData, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)
	blocked_id, err := queries.GetUserIDByTag(query_ctx, data.BlockedTag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	user_ctx, ok := ctx.Locals(middleware.USER_CONTEXT_KEY).(middleware.UserContext)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unknown user",
		})
	}
	err = queries.CreateUserRelationship(query_ctx, basepool.CreateUserRelationshipParams{
		User1ID:            user_ctx.UserID,
		User2ID:            blocked_id,
		RelationshipType:   basepool.RelationshipBlocked,
		RelationshipStatus: basepool.RelationshipStatusAccepted,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot block this user",
		})
	}

	return ctx.SendStatus(fiber.StatusAccepted)
}

type UnblockUserData struct {
	BlockedTag string `json:"blocked_tag"`
}

func UnblockUserHandler(data UnblockUserData, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)
	blocked_id, err := queries.GetUserIDByTag(query_ctx, data.BlockedTag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	user_ctx, ok := ctx.Locals(middleware.USER_CONTEXT_KEY).(middleware.UserContext)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unknown user",
		})
	}
	err = queries.DelUserRelationhip(query_ctx, basepool.DelUserRelationhipParams{
		User1ID:          user_ctx.UserID,
		User2ID:          blocked_id,
		RelationshipType: basepool.RelationshipBlocked,
	})
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot unblock this user",
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
	friend_id, err := queries.GetUserIDByTag(query_ctx, data.FriendTag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	user_ctx, ok := ctx.Locals(middleware.USER_CONTEXT_KEY).(middleware.UserContext)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	err = queries.DelUserRelationhip(query_ctx, basepool.DelUserRelationhipParams{
		User1ID:          user_ctx.UserID,
		User2ID:          friend_id,
		RelationshipType: basepool.RelationshipFriend,
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
	friend_id, err := queries.GetUserIDByTag(query_ctx, data.FriendTag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	user_ctx, ok := ctx.Locals(middleware.USER_CONTEXT_KEY).(middleware.UserContext)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	err = queries.DelPendingRequest(query_ctx, basepool.DelPendingRequestParams{
		User1ID: user_ctx.UserID,
		User2ID: friend_id,
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

func FriendResponceHandler(data FriendResponseData, ctx *fiber.Ctx, db *services.Database) error {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)
	requester_id, err := queries.GetUserIDByTag(query_ctx, data.RequesterTag)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot find this user",
		})
	}

	user_ctx, ok := ctx.Locals(middleware.USER_CONTEXT_KEY).(middleware.UserContext)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unknown user",
		})
	}
	var status basepool.RelationshipStatus
	if data.Response {
		status = basepool.RelationshipStatusAccepted
	} else {
		status = basepool.RelationshipStatusRejected
	}
	err = queries.UpdateUserRelationshipStatus(query_ctx, basepool.UpdateUserRelationshipStatusParams{
		User1ID:            requester_id,
		User2ID:            user_ctx.UserID,
		RelationshipStatus: status,
	})

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

	user_ctx, ok := ctx.Locals(middleware.USER_CONTEXT_KEY).(middleware.UserContext)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	friends, err := queries.GetUserFriends(query_ctx, user_ctx.UserID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "unknown user",
		})
	}

	return ctx.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"friends": friends,
	})
}
