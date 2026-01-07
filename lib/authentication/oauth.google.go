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

type GoogleOAuthProvider struct {
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
}

func NewGoogleOAuthProvider(clientID, clientSecret, redirectURI string) *GoogleOAuthProvider {
	return &GoogleOAuthProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
		},
	}
}

func (g *GoogleOAuthProvider) GetAuthURL(state string) string {
	baseURL := "https://accounts.google.com/o/oauth2/v2/auth"
	params := url.Values{
		"client_id":     {g.clientID},
		"redirect_uri":  {g.redirectURI},
		"response_type": {"code"},
		"scope":         {strings.Join(g.scopes, " ")},
		"state":         {state},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
	}
	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

func (g *GoogleOAuthProvider) ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	tokenURL := "https://oauth2.googleapis.com/token"
	data := url.Values{
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"code":          {code},
		"redirect_uri":  {g.redirectURI},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
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

	tokens.ExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	return &tokens, nil
}

func (g *GoogleOAuthProvider) GetUserProfile(ctx context.Context, tokens *OAuthTokens) (*OAuthProfile, error) {
	userInfoURL := "https://www.googleapis.com/oauth2/v3/userinfo"
	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user info request failed: %s", string(body))
	}

	var googleProfile struct {
		Sub      string `json:"sub"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Picture  string `json:"picture"`
		Locale   string `json:"locale"`
		Verified bool   `json:"email_verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleProfile); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Create UUID from sub
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(googleProfile.Sub))
	var pgID pgtype.UUID
	pgID.Bytes = id
	pgID.Valid = true

	// Extract country from locale (e.g., "en-US" -> "us")
	country := "us" // default
	if len(googleProfile.Locale) >= 5 {
		country = strings.ToLower(googleProfile.Locale[3:5])
	}

	return &OAuthProfile{
		Provider:  basepool.AuthTypeGoogle,
		ID:        pgID,
		Email:     googleProfile.Email,
		Username:  googleProfile.Name,
		AvatarURL: googleProfile.Picture,
		Country:   country,
	}, nil
}

func (g *GoogleOAuthProvider) ValidateState(state string, cache *services.Cache) bool {
	val, err := cache.Db.Get(context.Background(), fmt.Sprintf("oauth_state:%s", state)).Result()
	if err != nil || val != "pending" {
		return false
	}
	cache.Db.Del(context.Background(), fmt.Sprintf("oauth_state:%s", state))
	return true
}
