package authentication

import (
	"backend/lib/services"
	"backend/lib/vault"
	"context"
	"errors"
	"fmt"
	"time"

	basepool "github.com/ciphrpool/base-pool/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

const DEFAULT_USER_ELO = 1000

// SignUp creates a new user account with OAuth profile
func (a *AuthService) SignUp(
	ctx context.Context,
	user_profile *basepool.CreateUserParams,
	oauth_profile *OAuthProfile,
	cache *services.Cache,
	db *services.Database,
	vault *vault.VaultManager,
) (*TokenPairWithUserInfo, error) {
	// Check if user already exists
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	existingAccount, err := queries.GetAuthAccountByID(query_ctx, oauth_profile.ID)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if existingAccount.UserID.Bytes != uuid.Nil {
		return nil, ErrUserExists
	}

	// Start transaction
	tx, err := db.Pool.Begin(query_ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(query_ctx)

	qtx := queries.WithTx(tx)

	// Create new user
	user_id, err := qtx.CreateUser(ctx, *user_profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	err = qtx.CreateUserDefaultSettings(ctx, user_id)
	if err != nil {
		return nil, fmt.Errorf("failed to create user default settings: %w", err)
	}

	// Store OAuth account info
	err = qtx.CreateAuthAccount(ctx, basepool.CreateAuthAccountParams{
		UserID:   user_id,
		Email:    oauth_profile.Email,
		AuthType: oauth_profile.Provider,
		AuthID:   oauth_profile.ID,
		Verified: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create auth account: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Generate tokens
	tokenPair, err := a.tokenService.GenerateTokenPair(ctx, user_id, cache)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &TokenPairWithUserInfo{
		TokenPair: *tokenPair,
		UserID:    user_id,
	}, nil
}

// SignIn authenticates an existing user
func (a *AuthService) SignIn(
	ctx context.Context,
	oauth_profile *OAuthProfile,
	cache *services.Cache,
	db *services.Database,
	vault *vault.VaultManager,
) (*TokenPairWithUserInfo, error) {
	query_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	queries := basepool.New(db.Pool)

	// Find user by OAuth provider and ID
	authAccount, err := queries.GetAuthAccountByID(query_ctx, oauth_profile.ID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if !authAccount.Verified {
		return nil, fmt.Errorf("user not verified")
	}

	// Update last login
	err = queries.UpdateLastLogin(ctx, authAccount.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to update login time: %w", err)
	}

	// Generate new tokens
	tokenPair, err := a.tokenService.GenerateTokenPair(ctx, authAccount.UserID, cache)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &TokenPairWithUserInfo{
		TokenPair: *tokenPair,
		UserID:    authAccount.UserID,
	}, nil
}

// SignOut invalidates user's tokens and sessions
func (a *AuthService) SignOut(
	ctx context.Context,
	userID pgtype.UUID,
	cache *services.Cache,
	db *services.Database,
	vault *vault.VaultManager,
) error {

	return nil
}

// LogOut invalidates user's tokens and sessions
func (a *AuthService) LogOut(
	ctx context.Context,
	userID pgtype.UUID,
	cache *services.Cache,
) error {
	// Revoke all tokens
	err := a.RevokeUserTokens(ctx, userID, cache)
	if err != nil {
		return fmt.Errorf("failed to revoke tokens: %w", err)
	}

	return nil
}
