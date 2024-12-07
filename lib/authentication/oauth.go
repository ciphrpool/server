package authentication

import (
	"backend/lib/services"
	"backend/lib/vault"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/gofiber/fiber/v2"
)

var (
	ErrProviderNotFound = errors.New("oauth provider not configured")
	ErrInvalidState     = errors.New("invalid oauth state")
	ErrCodeExchange     = errors.New("failed to exchange oauth code")
	ErrUserInfo         = errors.New("failed to get user info")
)

// generateState creates a secure random state parameter for OAuth
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// InitiateOAuth starts the OAuth flow by generating a state and getting the provider's auth URL
func (a *AuthService) InitiateOAuth(ctx context.Context, provider basepool.AuthType, cache *services.Cache) (string, error) {
	oauthProvider, exists := a.oauthProviders[provider]
	if !exists {
		return "", ErrProviderNotFound
	}

	// Generate secure state parameter
	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	err = cache.Db.Set(ctx,
		fmt.Sprintf("oauth_state:%s", state),
		"pending",
		5*time.Minute,
	).Err()
	if err != nil {
		return "", err
	}
	// Get authorization URL from provider
	authURL := oauthProvider.GetAuthURL(state)

	return authURL, nil
}

// HandleOAuthCallback processes the OAuth callback and returns tokens for the authenticated user
func (a *AuthService) HandleOAuthCallback(
	ctx *fiber.Ctx,
	provider basepool.AuthType,
	code string,
	state string,
	cache *services.Cache,
	db *services.Database,
	vault *vault.VaultManager,
) (*TokenPairWithUserInfo, error) {
	oauthProvider, exists := a.oauthProviders[provider]
	if !exists {
		return nil, ErrProviderNotFound
	}

	// Validate state parameter
	if !oauthProvider.ValidateState(state, cache) {
		return nil, ErrInvalidState
	}

	// Exchange code for OAuth tokens
	oauthTokens, err := oauthProvider.ExchangeCode(ctx.Context(), code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCodeExchange, err)
	}

	// Get user profile from provider
	oauthProfile, err := oauthProvider.GetUserProfile(ctx.Context(), oauthTokens)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserInfo, err)
	}

	// Generate application tokens

	// Try to sign in first (if user exists)
	tokenPair, err := a.SignIn(ctx.Context(), oauthProfile, cache, db, vault)
	if err != nil {
		// If user doesn't exist, create a new account
		if errors.Is(err, ErrUserNotFound) {
			// Create new user profile from OAuth data
			newUser := &basepool.CreateUserParams{
				Username:          oauthProfile.Username,
				Country:           oauthProfile.Country,
				ProfilePictureUrl: oauthProfile.AvatarURL,
				Elo:               DEFAULT_USER_ELO,
			}

			// Sign up new user
			tokenPair, err = a.SignUp(ctx.Context(), newUser, oauthProfile, cache, db, vault)
			if err != nil {
				return nil, fmt.Errorf("failed to create new user: %w", err)
			}
			return tokenPair, nil
		}
		return nil, fmt.Errorf("failed to sign in user: %w", err)
	}

	return tokenPair, nil
}
