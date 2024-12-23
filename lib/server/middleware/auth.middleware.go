package middleware

import (
	"backend/lib/authentication"
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrNoAuthHeader      = errors.New("no authorization header")
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")
	ErrInvalidToken      = errors.New("invalid token")
	ErrSessionCreation   = errors.New("failed to create session")
	ErrInvalidCSRF       = errors.New("invalid or missing CSRF token")
)

// extractBearerToken gets the token from Authorization header
func extractBearerToken(c *fiber.Ctx) (string, error) {
	auth := c.Get("Authorization")
	if auth == "" {
		return "", ErrNoAuthHeader
	}

	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ErrInvalidAuthHeader
	}

	return parts[1], nil
}

// Protected is the main authentication middleware
func Protected(auth **authentication.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract and validate CSRF token first
		csrf_token_cookie := c.Cookies("CSRF-TOKEN")
		if len(csrf_token_cookie) == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No CSRF-TOKEN in Cookies",
			})
		}
		csrf_token := c.Get("X-CSRF-Token")
		if csrf_token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No CSRF-TOKEN in Headers",
			})
		}

		if csrf_token_cookie != csrf_token {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Different CSRF tokens",
			})
		}
		// Extract token
		access_token, err := extractBearerToken(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Validate token
		claims, err := (*auth).ValidateUserToken(c.Context(), access_token, csrf_token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}
		slog.Debug("The access_token is valid")

		// Set user info and session in context
		c.Locals("userID", claims.UserID)

		return c.Next()
	}
}

// RequireSession ensures a valid session exists
func RequireSession(auth **authentication.AuthService, sessions *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {

		valid, sessionID, err := (*auth).ValidateSession(c, sessions)
		if err != nil || !valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired session",
			})
		}
		c.Locals("sessionID", sessionID)

		return c.Next()
	}
}

// GetUserID helper to get userID from context
func GetUserID(c *fiber.Ctx) (pgtype.UUID, error) {
	userID, ok := c.Locals("userID").(pgtype.UUID)
	if !ok {
		return pgtype.UUID{}, errors.New("user ID not found in context")
	}
	return userID, nil
}

// GetSessionId helper to get sessionID from context
func GetSessionId(c *fiber.Ctx) (string, error) {
	sessionID, ok := c.Locals("sessionID").(string)
	if !ok {
		return "", errors.New("user ID not found in context")
	}
	return sessionID, nil
}
