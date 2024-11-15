package middleware

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserContext struct {
	UserID pgtype.UUID
	IP     string
}

const USER_CONTEXT_KEY = "user_context"

func StringToUUID(s string) (pgtype.UUID, error) {
	var pgUUID pgtype.UUID

	// Parse the string into a UUID
	parsedUUID, err := uuid.Parse(s)
	if err != nil {
		return pgUUID, err
	}

	// Convert to pgtype.UUID
	pgUUID.Bytes = parsedUUID
	pgUUID.Valid = true

	return pgUUID, nil
}

func UUIDToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}

func ForAuthentificatedUser(jwt_key func() (string, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		jwt_key, err := jwt_key()
		if err != nil {
			slog.Error(err.Error())
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "cannot access jwt key",
			})
		}

		auth_header := c.Get("Authorization")
		if auth_header == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		token_str := strings.TrimPrefix(auth_header, "Bearer ")
		if token_str == auth_header {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		token, err := jwt.Parse(token_str, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwt_key), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}
		user_id, err := StringToUUID(claims["user_id"].(string))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}
		user_ctx := UserContext{
			UserID: user_id,
			IP:     c.IP(),
		}

		c.Locals(USER_CONTEXT_KEY, user_ctx)

		return c.Next()
	}
}
