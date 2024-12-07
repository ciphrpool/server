package authentication

import (
	"backend/lib/services"
	"backend/lib/vault"
	"context"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type OAuthProfile struct {
	Provider  basepool.AuthType `json:"provider"`
	ID        pgtype.UUID       `json:"id"`
	Email     string            `json:"email"`
	Username  string            `json:"username"`
	AvatarURL string            `json:"avatar_url"`
	Country   string            `json:"country"`
}

type OAuthProvider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error)
	GetUserProfile(ctx context.Context, tokens *OAuthTokens) (*OAuthProfile, error)
	ValidateState(state string, cache *services.Cache) bool
}

type AuthService struct {
	tokenService   TokenService
	oauthProviders map[basepool.AuthType]OAuthProvider
}

type AuthServiceInterface interface {
	// OAuth Flow
	InitiateOAuth(ctx context.Context, provider basepool.AuthType, cache *services.Cache) (string, error)
	HandleOAuthCallback(ctx *fiber.Ctx, provider basepool.AuthType, code string, state string, cache *services.Cache, db *services.Database, vault *vault.VaultManager) (*TokenPairWithUserInfo, error)

	// User Authentication
	SignUp(ctx context.Context, user *basepool.CreateUserParams, profile *OAuthProfile, cache *services.Cache, db *services.Database, vault *vault.VaultManager) (*TokenPairWithUserInfo, error)
	SignIn(ctx context.Context, profile *OAuthProfile, cache *services.Cache, db *services.Database, vault *vault.VaultManager) (*TokenPairWithUserInfo, error)
	SignOut(ctx context.Context, userID uuid.UUID, cache *services.Cache, db *services.Database, vault *vault.VaultManager) error
	LogOut(ctx context.Context, userID uuid.UUID, cache *services.Cache) error

	// Token Management
	RefreshUserAccessToken(ctx context.Context, csrf_token string, refreshToken string, cache *services.Cache) (string, time.Time, error)
	RefreshUserTokens(ctx context.Context, refreshToken string, cache *services.Cache) (*TokenPair, error)
	RevokeUserTokens(ctx context.Context, userID uuid.UUID, cache *services.Cache) error
	ValidateUserToken(ctx context.Context, access_token string, csrf_token string) (*Claims, error)
	ValidateUserRefreshToken(ctx context.Context, refreshToken string, cache *services.Cache) (*Claims, error)

	// Session Management
	CreateSession(ctx *fiber.Ctx, userID uuid.UUID, sessions *session.Store) (string, error)
	ValidateSession(ctx *fiber.Ctx, sessionID string, sessions *session.Store) (bool, error)
	DestroySession(ctx *fiber.Ctx, sessionID string, sessions *session.Store) error
}

type AuthConfig struct {
	TokenConfig TokenConfig
	OAuthConfig OAuthConfig
}
