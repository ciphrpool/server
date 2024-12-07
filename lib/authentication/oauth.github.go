package authentication

import (
	"backend/lib/services"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type GithubOAuthProvider struct {
	clientID     string
	clientSecret string
	redirectURI  string
}

func NewGithubOAuthProvider(clientID, clientSecret, redirectURI string) *GithubOAuthProvider {
	return &GithubOAuthProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
	}
}

func (g *GithubOAuthProvider) GetAuthURL(state string) string {
	baseURL := "https://github.com/login/oauth/authorize"
	params := url.Values{
		"client_id":    {g.clientID},
		"redirect_uri": {g.redirectURI},
		"scope":        {"user:email read:user"},
		"state":        {state},
	}
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

func (g *GithubOAuthProvider) ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	tokenURL := "https://github.com/login/oauth/access_token"
	data := url.Values{
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"code":          {code},
		"redirect_uri":  {g.redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokens OAuthTokens
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	tokens.ExpiresAt = time.Now().Add(time.Hour * 24)
	return &tokens, nil
}

func (g *GithubOAuthProvider) GetUserProfile(ctx context.Context, tokens *OAuthTokens) (*OAuthProfile, error) {
	// Get user info
	userProfile, err := g.getUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		return nil, err
	}

	// Get primary email if not available in profile
	if userProfile.Email == "" {
		email, err := g.getPrimaryEmail(ctx, tokens.AccessToken)
		if err != nil {
			return nil, err
		}
		userProfile.Email = email
	}

	return userProfile, nil
}

func (g *GithubOAuthProvider) getUserInfo(ctx context.Context, accessToken string) (*OAuthProfile, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var githubProfile struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Location  string `json:"location"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&githubProfile); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Create UUID from GitHub ID
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("github:%d", githubProfile.ID)))
	var pgID pgtype.UUID
	pgID.Bytes = id
	pgID.Valid = true

	// Default username to login if name is empty
	username := githubProfile.Name
	if username == "" {
		username = githubProfile.Login
	}

	// Try to extract country from location (simplified)
	country := "us" // default
	if githubProfile.Location != "" {
		country = g.extractCountryCode(githubProfile.Location)
	}

	return &OAuthProfile{
		Provider:  basepool.AuthTypeGithub,
		ID:        pgID,
		Email:     githubProfile.Email,
		Username:  username,
		AvatarURL: githubProfile.AvatarURL,
		Country:   country,
	}, nil
}

func (g *GithubOAuthProvider) getPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	return "", fmt.Errorf("no primary verified email found")
}

func (g *GithubOAuthProvider) extractCountryCode(location string) string {
	// This is a simplified version. In a production environment,
	// you might want to use a more sophisticated location -> country code mapping
	location = strings.ToLower(location)
	countryMap := map[string]string{
		"united states": "us",
		"usa":           "us",
		"uk":            "gb",
		"france":        "fr",
		"germany":       "de",
		// Add more mappings as needed
	}

	for key, code := range countryMap {
		if strings.Contains(location, key) {
			return code
		}
	}
	return "us"
}

func (g *GithubOAuthProvider) ValidateState(state string, cache *services.Cache) bool {
	val, err := cache.Db.Get(context.Background(), fmt.Sprintf("oauth_state:%s", state)).Result()
	if err != nil || val != "pending" {
		return false
	}
	cache.Db.Del(context.Background(), fmt.Sprintf("oauth_state:%s", state))
	return true
}
