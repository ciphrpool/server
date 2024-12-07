//go:build dev

package authentication

import (
	"backend/lib/vault"
	"fmt"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
)

type MockServerConfig struct {
	ServerURL    string
	ClientID     string
	ClientSecret string
	RedirectURI  string
}
type OAuthConfig struct {
	MockServerConfig MockServerConfig
}

func BuildAuthConfig(vault *vault.VaultManager) (*AuthConfig, error) {
	jwt_key, err := vault.GetApiKey("JWT_KEY")
	if err != nil {
		return nil, fmt.Errorf("unavailable jwt_key")
	}

	return &AuthConfig{
		TokenConfig: TokenConfig{
			SigningKey:      jwt_key,
			TokenDuration:   30 * time.Minute,
			RefreshDuration: 7 * 24 * time.Hour, // 7 days
		},
		OAuthConfig: OAuthConfig{
			MockServerConfig: MockServerConfig{
				ServerURL:    "http://172.17.0.1:3202",
				RedirectURI:  "http://localhost:3000/auth/callback/google",
				ClientID:     "MOCK_CLIENT_ID",
				ClientSecret: "MOCK_CLIENT_SECRET",
			},
		},
	}, nil
}

func NewAuthService(config *AuthConfig) (*AuthService, error) {
	// Initialize token service
	tokenService := NewJWTTokenService(config.TokenConfig)

	// Initialize OAuth providers
	oauthProviders := make(map[basepool.AuthType]OAuthProvider)

	mockProvider := NewMockOAuthProvider(
		config.OAuthConfig.MockServerConfig.ClientID,
		config.OAuthConfig.MockServerConfig.ClientSecret,
		config.OAuthConfig.MockServerConfig.RedirectURI,
		config.OAuthConfig.MockServerConfig.ServerURL,
	)
	oauthProviders[basepool.AuthTypeGoogle] = mockProvider

	// Create new auth service
	authService := &AuthService{
		tokenService:   tokenService,
		oauthProviders: oauthProviders,
	}

	return authService, nil
}
