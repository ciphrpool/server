//go:build !dev

package authentication

import (
	"backend/lib/vault"
	"fmt"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
)

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}
type GithubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}
type OAuthConfig struct {
	GoogleConfig GoogleConfig
	GithubConfig GithubConfig
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
		OAuthConfig: OAuthConfig{},
	}, nil
}

func NewAuthService(config *AuthConfig) (*AuthService, error) {
	// Initialize token service
	tokenService := NewJWTTokenService(config.TokenConfig)

	// Initialize OAuth providers
	oauthProviders := make(map[basepool.AuthType]OAuthProvider)

	// Initialize real providers
	googleProvider := NewGoogleOAuthProvider(
		config.OAuthConfig.GoogleConfig.ClientID,
		config.OAuthConfig.GoogleConfig.ClientSecret,
		config.OAuthConfig.GoogleConfig.RedirectURI,
	)
	githubProvider := NewGithubOAuthProvider(
		config.OAuthConfig.GithubConfig.ClientID,
		config.OAuthConfig.GithubConfig.ClientSecret,
		config.OAuthConfig.GithubConfig.RedirectURI,
	)

	oauthProviders[basepool.AuthTypeGoogle] = googleProvider
	oauthProviders[basepool.AuthTypeGithub] = githubProvider

	// Create new auth service
	authService := &AuthService{
		tokenService:   tokenService,
		oauthProviders: oauthProviders,
	}

	return authService, nil
}
