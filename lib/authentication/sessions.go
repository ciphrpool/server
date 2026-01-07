package authentication

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	sessionIDPrefix = "session:"
	sessionDuration = 24 * time.Hour // Default session duration
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionInvalid  = errors.New("session is invalid")
	ErrSessionExpired  = errors.New("session has expired")
)

type SessionData struct {
	UserID    pgtype.UUID `json:"user_id"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt time.Time   `json:"expires_at"`
	LastSeen  time.Time   `json:"last_seen"`
	UserAgent string      `json:"user_agent"`
	IPAddress string      `json:"ip_address"`
}

// CreateSession creates a new session for a user
func (a *AuthService) CreateSession(
	ctx *fiber.Ctx,
	userID pgtype.UUID,
	sessions *session.Store,
) (string, error) {
	// Create session data
	sessionData := SessionData{
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sessionDuration),
		LastSeen:  time.Now(),
	}

	// add more security data
	sessionData.UserAgent = ctx.Get("User-Agent")
	sessionData.IPAddress = ctx.IP()

	// Store session in Fiber session store
	sess, err := sessions.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	// Store the complete session data
	sessionDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Store session data
	sess.Set("data", string(sessionDataJSON))

	// Set session expiry
	sess.SetExpiry(sessionDuration)
	id := sess.ID()
	// Save session
	if err := sess.Save(); err != nil {
		return "", fmt.Errorf("failed to save session: %w", err)
	}
	return id, nil
}

// ValidateSession validates a session and updates last seen time
func (a *AuthService) ValidateSession(
	ctx *fiber.Ctx,
	sessions *session.Store,
) (bool, string, error) {
	// Get session from store
	sess, err := sessions.Get(ctx)
	if err != nil {
		return false, "", fmt.Errorf("failed to get session: %w", err)
	}
	sessionId := sess.ID()
	// Get session data
	sessionDataStr := sess.Get("data")
	if sessionDataStr == nil {
		return false, "", nil
	}

	var sessionData SessionData
	err = json.Unmarshal([]byte(sessionDataStr.(string)), &sessionData)
	if err != nil {
		return false, "", fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// Check if session is expired
	if time.Now().After(sessionData.ExpiresAt) {
		// Clean up expired session
		if err := a.DestroySession(ctx, sess.ID(), sessions); err != nil {
			return false, "", fmt.Errorf("failed to clean up expired session: %w", err)
		}
		return false, "", ErrSessionExpired
	}

	// Validate User-Agent hasn't changed dramatically
	if sessionData.UserAgent != "" && sessionData.UserAgent != ctx.Get("User-Agent") {
		return false, "", ErrSessionInvalid
	}

	// Validate IP hasn't changed dramatically
	if sessionData.IPAddress != "" && sessionData.IPAddress != ctx.IP() {
		return false, "", ErrSessionInvalid
	}
	// Update last seen time
	sessionData.LastSeen = time.Now()

	// Save updated session data
	updatedDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return false, "", fmt.Errorf("failed to marshal updated session data: %w", err)
	}

	sess.Set("data", string(updatedDataJSON))

	if err := sess.Save(); err != nil {
		return false, "", fmt.Errorf("failed to save updated session: %w", err)
	}

	return true, sessionId, nil
}

// DestroySession removes a session
func (a *AuthService) DestroySession(
	ctx *fiber.Ctx,
	sessionID string,
	sessions *session.Store,
) error {
	// Get session from store
	sess, err := sessions.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if sess.ID() != sessionID {
		return ErrSessionNotFound
	}

	// Destroy the session
	if err := sess.Destroy(); err != nil {
		return fmt.Errorf("failed to destroy session: %w", err)
	}

	return nil
}
