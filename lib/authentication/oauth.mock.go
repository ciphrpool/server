//go:build dev

package authentication

import (
	"backend/lib/services"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type MockOAuthProvider struct {
	clientID     string
	clientSecret string
	redirectURI  string
	serverURL    string // URL of the mock Python OAuth server
}

func NewMockOAuthProvider(clientID, clientSecret, redirectURI, serverURL string) *MockOAuthProvider {
	return &MockOAuthProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		serverURL:    serverURL,
	}
}

func (m *MockOAuthProvider) GetAuthURL(state string) string {
	params := url.Values{
		"client_id":     {m.clientID},
		"redirect_uri":  {m.redirectURI},
		"response_type": {"code"},
		"state":         {state},
		"scope":         {"email profile"},
	}
	return fmt.Sprintf("%s/authorize?%s", m.serverURL, params.Encode())
}

func (m *MockOAuthProvider) ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	data := url.Values{
		"client_id":     {m.clientID},
		"client_secret": {m.clientSecret},
		"code":          {code},
		"redirect_uri":  {m.redirectURI},
		"grant_type":    {"authorization_code"},
	}
	slog.Debug(fmt.Sprintf("%s/token", m.serverURL))
	slog.Debug("Request Body", "data", data.Encode())
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/token", m.serverURL), strings.NewReader(data.Encode()))
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

func (m *MockOAuthProvider) GetUserProfile(ctx context.Context, tokens *OAuthTokens) (*OAuthProfile, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/userinfo", m.serverURL), nil)
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

	var mockUser struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		Country   string `json:"country"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&mockUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(mockUser.ID))
	var pgID pgtype.UUID
	pgID.Bytes = id
	pgID.Valid = true

	return &OAuthProfile{
		Provider:  basepool.AuthTypeGoogle,
		ID:        pgID,
		Email:     mockUser.Email,
		Username:  mockUser.Username,
		AvatarURL: mockUser.AvatarURL,
		Country:   mockUser.Country,
	}, nil
}

func (m *MockOAuthProvider) ValidateState(state string, cache *services.Cache) bool {
	val, err := cache.Db.Get(context.Background(), fmt.Sprintf("oauth_state:%s", state)).Result()
	if err != nil || val != "pending" {
		return false
	}
	cache.Db.Del(context.Background(), fmt.Sprintf("oauth_state:%s", state))
	return true
}
