package routes

import (
	"backend/lib/authentication"
	"backend/lib/server/middleware"
	"backend/lib/services"
	"backend/lib/vault"
	"fmt"

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

	// return c.Redirect(auth_url)
	return c.JSON(fiber.Map{
		"auth_url": auth_url,
	})
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
	_, err = auth.CreateSession(c, token_pair_with_user_info.UserID, sessions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}

	// Set Refresh Token as HTTP-Only secure cookie
	c.Cookie(&fiber.Cookie{
		Name:     "REFRESH-TOKEN",
		Value:    token_pair_with_user_info.TokenPair.RefreshToken,
		Secure:   true,
		HTTPOnly: true,
		SameSite: "Strict",
		Path:     "/",
	})

	// Set CSRF Token as regular cookie (accessible by JavaScript)
	c.Cookie(&fiber.Cookie{
		Name:     "CSRF-TOKEN",
		Value:    token_pair_with_user_info.TokenPair.CSRFToken,
		Secure:   true,
		SameSite: "Strict",
		Path:     "/",
	})

	// Return access token and expiry
	return c.JSON(fiber.Map{
		"access_token": token_pair_with_user_info.TokenPair.AccessToken,
		"expires_at":   token_pair_with_user_info.TokenPair.ExpiresAt,
	})
}
func RefreshAccessTokenHandler(c *fiber.Ctx, auth *authentication.AuthService, cache *services.Cache) error {
	csrf_token_cookie := c.Cookies("CSRF-TOKEN")
	if len(csrf_token_cookie) == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": middleware.ErrInvalidCSRF.Error(),
		})
	}
	csrf_token := c.Get("X-CSRF-Token")
	if csrf_token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": middleware.ErrInvalidCSRF.Error(),
		})
	}
	if csrf_token_cookie != csrf_token {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": middleware.ErrInvalidCSRF.Error(),
		})
	}

	refresh_token_cookie := c.Cookies("REFRESH-TOKEN")
	if len(refresh_token_cookie) == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "no refresh token",
		})
	}

	access_token, expiresAt, err := auth.RefreshUserAccessToken(c.Context(), refresh_token_cookie, csrf_token, cache)
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
	refresh_token_cookie := c.Cookies("REFRESH-TOKEN")
	if len(refresh_token_cookie) == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "no refresh token",
		})
	}
	// Get claims from refresh token
	claims, err := auth.ValidateUserRefreshToken(c.Context(), refresh_token_cookie, cache)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}

	_, err = auth.CreateSession(c, claims.UserID, sessions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}

	return c.SendStatus(fiber.StatusOK)
}

func RefreshAllTokenHandler(c *fiber.Ctx, auth *authentication.AuthService, cache *services.Cache) error {

	refresh_token_cookie := c.Cookies("REFRESH-TOKEN")
	if len(refresh_token_cookie) == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "no refresh token",
		})
	}

	token_pair, err := auth.RefreshUserTokens(c.Context(), refresh_token_cookie, cache)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid refresh token",
		})
	}

	// Set Refresh Token as HTTP-Only secure cookie
	c.Cookie(&fiber.Cookie{
		Name:     "REFRESH-TOKEN",
		Value:    token_pair.RefreshToken,
		Secure:   true,
		HTTPOnly: true,
		SameSite: "Strict",
		Path:     "/",
	})

	// Set CSRF Token as regular cookie (accessible by JavaScript)
	c.Cookie(&fiber.Cookie{
		Name:     "CSRF-TOKEN",
		Value:    token_pair.CSRFToken,
		Secure:   true,
		SameSite: "Strict",
		Path:     "/",
	})
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token": token_pair.AccessToken,
		"expires_at":   token_pair.ExpiresAt,
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
	return c.SendStatus(fiber.StatusOK)
}
