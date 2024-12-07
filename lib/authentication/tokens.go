package authentication

import (
	"backend/lib/services"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	math_rand "math/rand"
	"reflect"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	CSRFToken    string    `json:"csrf_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type TokenPairWithUserInfo struct {
	TokenPair TokenPair
	UserID    pgtype.UUID `json:"user_id"`
}

type Claims struct {
	UserID    pgtype.UUID `json:"user_id"`
	Username  string      `json:"username"`
	ExpiresAt int64       `json:"exp"`
}

type OAuthTokens struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int       `json:"expires_in"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type TokenService interface {
	GenerateTokenPair(ctx context.Context, userID pgtype.UUID, cache *services.Cache) (*TokenPair, error)
	ValidateAccessToken(ctx context.Context, access_token string, csrf_token string) (*Claims, error)
	ValidateRefreshToken(ctx context.Context, token string, cache *services.Cache) (*Claims, error)
	RevokeTokens(ctx context.Context, userID pgtype.UUID, cache *services.Cache) error
	RefreshTokens(ctx context.Context, userID pgtype.UUID, refreshToken string, cache *services.Cache) (*TokenPair, error)
	RefreshAccessTokens(ctx context.Context, userID pgtype.UUID, crsf_token string, refreshToken string, cache *services.Cache) (string, time.Time, error)
}

const (
	accessTokenPrefix   = "access_token:"
	refreshTokenPrefix  = "refresh_token:"
	revokedTokenPrefix  = "revoked_token:"
	tokenDurationBuffer = 5 * time.Minute // Buffer time for token operations
)

var (
	ErrTokenExpired      = errors.New("token has expired")
	ErrTokenRevoked      = errors.New("token has been revoked")
	ErrTokenInvalid      = errors.New("token is invalid")
	ErrTokenNotFound     = errors.New("token not found")
	ErrRefreshTokenUsed  = errors.New("refresh token has already been used")
	ErrInvalidToken      = errors.New("invalid token")
	ErrCSRFTokenMismatch = errors.New("csrf token mismatch")
)

type CustomClaims struct {
	jwt.RegisteredClaims
	UserID   string `json:"user_id"`
	CSRFHash string `json:"csrf_hash"`
}

type JWTTokenService struct {
	signingKey      []byte
	tokenDuration   time.Duration
	refreshDuration time.Duration
}
type TokenConfig struct {
	SigningKey      string
	TokenDuration   time.Duration
	RefreshDuration time.Duration
}

func NewJWTTokenService(config TokenConfig) *JWTTokenService {
	return &JWTTokenService{
		signingKey:      []byte(config.SigningKey),
		tokenDuration:   config.TokenDuration,
		refreshDuration: config.RefreshDuration,
	}
}

// GenerateTokenPair creates a new pair of access and refresh tokens
func (s *JWTTokenService) GenerateTokenPair(ctx context.Context, userID pgtype.UUID, cache *services.Cache) (*TokenPair, error) {
	// Generate CSRF token
	csrfToken, err := s.generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate csrf token: %w", err)
	}

	// Calculate CSRF hash
	csrfHash := s.hashToken(csrfToken)

	// Create access token
	jitter := time.Duration(math_rand.Int63n(int64(30 * time.Second)))
	expiresAt := time.Now().Add(s.tokenDuration).Add(jitter)

	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		UserID:   services.UUIDToString(userID),
		CSRFHash: csrfHash,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(s.signingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateSecureToken(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in Redis
	refreshClaims := map[string]interface{}{
		"user_id":    services.UUIDToString(userID),
		"csrf_hash":  csrfHash,
		"created_at": time.Now().Unix(),
	}

	refreshClaimsJSON, err := json.Marshal(refreshClaims)
	slog.Debug("GENERATE CLAIMS", "time.Now().Unix()", time.Now().Unix())
	slog.Debug("GENERATE CLAIMS", "time.Now().Unix() type", reflect.TypeOf(time.Now().Unix()))
	slog.Debug("GENERATE CLAIMS", "refreshClaimsJSON", refreshClaimsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refresh claims: %w", err)
	}

	err = cache.Db.Set(ctx,
		fmt.Sprintf("refresh_token:%s", refreshToken),
		refreshClaimsJSON,
		s.refreshDuration,
	).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		CSRFToken:    csrfToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// ValidateAccessToken validates the access token and CSRF token
func (s *JWTTokenService) ValidateAccessToken(ctx context.Context, access_token string, csrf_token string) (*Claims, error) {
	// Parse and validate JWT
	token, err := jwt.ParseWithClaims(access_token, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.signingKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	// Verify CSRF token
	if s.hashToken(csrf_token) != claims.CSRFHash {
		return nil, ErrCSRFTokenMismatch
	}

	user_id, err := services.StringToUUID(claims.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	// Convert to generic Claims struct
	return &Claims{
		UserID:    user_id,
		ExpiresAt: claims.ExpiresAt.Unix(),
	}, nil
}

// ValidateRefreshToken validates the refresh token
func (s *JWTTokenService) ValidateRefreshToken(ctx context.Context, refreshToken string, cache *services.Cache) (*Claims, error) {
	// Get refresh token data from Redis
	data, err := cache.Db.Get(ctx, fmt.Sprintf("refresh_token:%s", refreshToken)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	var claims map[string]interface{}
	err = json.Unmarshal([]byte(data), &claims)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}
	user_id, err := services.StringToUUID(claims["user_id"].(string))
	if err != nil {
		return nil, ErrInvalidToken
	}
	// Convert to Claims struct
	slog.Debug("CLAIMS IN VALIDATE REFRESH TOKEN", "created_at", int64(math.Round(claims["created_at"].(float64))))
	slog.Debug("CLAIMS IN VALIDATE REFRESH TOKEN", "created_at_type", reflect.TypeOf(claims["created_at"]))
	return &Claims{
		UserID:    user_id,
		ExpiresAt: int64(math.Round(claims["created_at"].(float64))) + int64(math.Round(s.refreshDuration.Seconds())),
	}, nil
}

// RefreshTokens generates new token pair using refresh token
func (s *JWTTokenService) RefreshTokens(ctx context.Context, userID pgtype.UUID, refreshToken string, cache *services.Cache) (*TokenPair, error) {
	// Delete old refresh token
	err := cache.Db.Del(ctx, fmt.Sprintf("refresh_token:%s", refreshToken)).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to delete old refresh token: %w", err)
	}

	// Generate new token pair
	return s.GenerateTokenPair(ctx, userID, cache)
}

func (s *JWTTokenService) RefreshAccessTokens(ctx context.Context, userID pgtype.UUID, crsf_token string, refreshToken string, cache *services.Cache) (string, time.Time, error) {

	// Calculate CSRF hash
	csrfHash := s.hashToken(crsf_token)

	// Create access token
	jitter := time.Duration(math_rand.Int63n(int64(30 * time.Second)))
	expiresAt := time.Now().Add(s.tokenDuration).Add(jitter)

	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		UserID:   services.UUIDToString(userID),
		CSRFHash: csrfHash,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(s.signingKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}
	return accessToken, expiresAt, nil
}

// RevokeTokens invalidates all tokens for a user
func (s *JWTTokenService) RevokeTokens(ctx context.Context, userID pgtype.UUID, cache *services.Cache) error {
	// Add user ID to revocation list with current timestamp
	err := cache.Db.Set(ctx,
		fmt.Sprintf("revoked:%s", services.UUIDToString(userID)),
		time.Now().Unix(),
		s.refreshDuration,
	).Err()
	if err != nil {
		return fmt.Errorf("failed to revoke tokens: %w", err)
	}
	return nil
}

// Helper functions

func (s *JWTTokenService) generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (s *JWTTokenService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// AUTH SERVICE

// RefreshUserTokens generates new token pair using a valid refresh token
func (a *AuthService) RefreshUserAccessToken(
	ctx context.Context,
	refreshToken string,
	csrf_token string,
	cache *services.Cache,
) (string, time.Time, error) {
	// Get claims from refresh token
	claims, err := a.tokenService.ValidateRefreshToken(ctx, refreshToken, cache)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if user's tokens have been revoked
	revokedKey := fmt.Sprintf("%s%s", revokedTokenPrefix, services.UUIDToString(claims.UserID))
	revocationTime, err := cache.Db.Get(ctx, revokedKey).Result()
	if err == nil {
		// Revocation timestamp exists, check if token was issued before revocation
		var timestamp int64
		err := json.Unmarshal([]byte(revocationTime), &timestamp)
		if err == nil && claims.ExpiresAt < timestamp {
			return "", time.Time{}, ErrTokenRevoked
		}
	}

	// Generate new token pair
	return a.tokenService.RefreshAccessTokens(ctx, claims.UserID, csrf_token, refreshToken, cache)
}

// RefreshUserTokens generates new token pair using a valid refresh token
func (a *AuthService) RefreshUserTokens(
	ctx context.Context,
	refreshToken string,
	cache *services.Cache,
) (*TokenPair, error) {
	// Get claims from refresh token
	claims, err := a.tokenService.ValidateRefreshToken(ctx, refreshToken, cache)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if user's tokens have been revoked
	revokedKey := fmt.Sprintf("%s%s", revokedTokenPrefix, services.UUIDToString(claims.UserID))
	revocationTime, err := cache.Db.Get(ctx, revokedKey).Result()
	if err == nil {
		// Revocation timestamp exists, check if token was issued before revocation
		var timestamp int64
		err := json.Unmarshal([]byte(revocationTime), &timestamp)
		if err == nil && claims.ExpiresAt < timestamp {
			return nil, ErrTokenRevoked
		}
	}

	// Generate new token pair
	return a.tokenService.RefreshTokens(ctx, claims.UserID, refreshToken, cache)
}

// RevokeUserTokens invalidates all tokens for a user
func (a *AuthService) RevokeUserTokens(
	ctx context.Context,
	userID pgtype.UUID,
	cache *services.Cache,
) error {
	// First revoke tokens in the token service (handles refresh tokens)
	err := a.tokenService.RevokeTokens(ctx, userID, cache)
	if err != nil {
		return fmt.Errorf("failed to revoke tokens in token service: %w", err)
	}

	// Store revocation timestamp in cache
	revokedKey := fmt.Sprintf("%s%s", revokedTokenPrefix, services.UUIDToString(userID))
	timestamp := time.Now().Unix()

	// Store as json for consistency with other stored values
	timestampJSON, err := json.Marshal(timestamp)
	if err != nil {
		return fmt.Errorf("failed to marshal timestamp: %w", err)
	}

	err = cache.Db.Set(ctx, revokedKey, timestampJSON, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to store revocation timestamp: %w", err)
	}

	return nil
}

// ValidateUserToken validates an access token and returns its claims
func (a *AuthService) ValidateUserToken(
	ctx context.Context,
	access_token string,
	csrf_token string,
) (*Claims, error) {
	// Validate token cryptographically
	claims, err := a.tokenService.ValidateAccessToken(ctx, access_token, csrf_token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return claims, nil
}

func (a *AuthService) ValidateUserRefreshToken(
	ctx context.Context,
	refreshToken string,
	cache *services.Cache,
) (*Claims, error) {
	// Validate token cryptographically
	claims, err := a.tokenService.ValidateRefreshToken(ctx, refreshToken, cache)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return claims, nil
}
