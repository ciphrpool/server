package routes

import (
	"backend/lib/authentication"
	"backend/lib/server/middleware"
	"backend/lib/services"
	"backend/lib/vault"
	"fmt"
	"log/slog"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

func InitiateOAuthHandler(provider basepool.AuthType, c *fiber.Ctx, auth *authentication.AuthService, cache *services.Cache) error {
	auth_url, err := auth.InitiateOAuth(
		c.Context(),
		provider,
		cache,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to initiate OAuth flow",
		})
	}

	return c.Redirect(auth_url)
}

type OAuthCallbackParams struct {
	Code  string `query:"code"`
	State string `query:"state"`
}

func OAuthCallbackHandler(
	provider basepool.AuthType,
	params OAuthCallbackParams,
	c *fiber.Ctx,
	auth *authentication.AuthService,
	cache *services.Cache,
	db *services.Database,
	vault *vault.VaultManager,
	sessions *session.Store,
) error {
	if params.Code == "" || params.State == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing code or state parameter",
		})
	}

	// Handle OAuth callback and get tokens
	token_pair_with_user_info, err := auth.HandleOAuthCallback(
		c,
		provider,
		params.Code,
		params.State,
		cache,
		db,
		vault,
	)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Create session
	session_id, err := auth.CreateSession(c, token_pair_with_user_info.UserID, sessions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}

	c.Set("X-CSRF-Token", token_pair_with_user_info.TokenPair.CSRFToken)
	c.Set("X-Refresh-Token", token_pair_with_user_info.TokenPair.RefreshToken)
	slog.Debug("SESSION ID", "session id", session_id)
	c.Set("X-Session-ID", session_id)

	// Return access token and expiry
	return c.JSON(fiber.Map{
		"access_token": token_pair_with_user_info.TokenPair.AccessToken,
		"expires_at":   token_pair_with_user_info.TokenPair.ExpiresAt,
	})
}
func RefreshAccessTokenHandler(c *fiber.Ctx, auth *authentication.AuthService, cache *services.Cache) error {
	csrf_token := c.Get("X-CSRF-Token")
	if csrf_token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": middleware.ErrInvalidCSRF.Error(),
		})
	}

	refreshToken := c.Get("X-Refresh-Token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No refresh token provided",
		})
	}

	access_token, expiresAt, err := auth.RefreshUserAccessToken(c.Context(), csrf_token, refreshToken, cache)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid refresh token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token": access_token,
		"expires_at":   expiresAt,
	})
}

func RefreshSessionHandler(c *fiber.Ctx, auth *authentication.AuthService, cache *services.Cache, sessions *session.Store) error {
	refreshToken := c.Get("X-Refresh-Token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No refresh token provided",
		})
	}
	// Get claims from refresh token
	claims, err := auth.ValidateUserRefreshToken(c.Context(), refreshToken, cache)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}

	session_id, err := auth.CreateSession(c, claims.UserID, sessions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}
	slog.Debug("SESSION ID", "session id", session_id)
	c.Set("X-Session-ID", session_id)

	return c.SendStatus(fiber.StatusAccepted)
}

func RefreshAllTokenHandler(c *fiber.Ctx, auth *authentication.AuthService, cache *services.Cache) error {

	refreshToken := c.Get("X-Refresh-Token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No refresh token provided",
		})
	}

	token_pair, err := auth.RefreshUserTokens(c.Context(), refreshToken, cache)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid refresh token",
		})
	}

	c.Set("X-CSRF-Token", token_pair.CSRFToken)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token":  token_pair.AccessToken,
		"refresh_token": token_pair.RefreshToken,
		"expires_at":    token_pair.ExpiresAt,
	})
}

func LogoutHandler(c *fiber.Ctx, auth *authentication.AuthService, cache *services.Cache, sessions *session.Store) error {
	user_id, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user",
		})
	}
	sessionID, err := middleware.GetSessionId(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid session",
		})
	}
	err = auth.DestroySession(c, sessionID, sessions)
	if err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}

	err = auth.LogOut(c.Context(), user_id, cache)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user",
		})
	}
	return c.SendStatus(fiber.StatusAccepted)
}
