package notifications

import (
	"backend/lib/server/middleware"
	"backend/lib/services"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

// SSEHandler creates a Fiber handler for SSE connections
func (s *NotificationService) SSENotificationHandler(c *fiber.Ctx, cache *services.Cache) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}
	sessionId, err := middleware.GetSessionId(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown session",
		})
	}
	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// Create notification channel for this client
	notifications := make(chan *Notification, 100)

	// Register client
	if err := s.registerClient(c.Context(), userID, sessionId, notifications, cache); err != nil {
		slog.Error("failed to register client",
			"error", err,
			"user_id", services.UUIDToString(userID))
		return err
	}
	// Get buffer from pool
	bufPtr := s.bufPool.Get().(*[]byte)
	defer s.bufPool.Put(bufPtr)

	// Deliver any stored notifications
	if err := s.deliverStoredNotifications(c.Context(), userID, sessionId, cache); err != nil {
		slog.Error("Notifications : failed to deliver stored notifications",
			"error", err,
			"user_id", services.UUIDToString(userID))
	}

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Ensure client is unregistered when connection closes
		defer func() {
			s.unregisterClient(context.Background(), userID, sessionId, cache)
			// slog.Debug("Notifications : SSE notification session ended", "user_id", services.UUIDToString(userID))
			close(notifications)
		}()

		// Send initial connection established message
		w.WriteString("data: {\"type\":\"connected\"}\n\n")
		// slog.Debug("data: {\"type\":\"connected\"}")
		w.Flush()

		s.registry.mu.RLock()
		closeChan := s.registry.closeChans[sessionId]
		s.registry.mu.RUnlock()

		connectionCheckTicker := time.NewTicker(5 * time.Minute)
		defer connectionCheckTicker.Stop()

		// Start message loop
		for {
			select {
			case notification, ok := <-notifications:
				if !ok {
					return
				}
				slog.Debug("Notifications : Sending notification", "notification", notification)

				// Reset buffer and encode notification
				buf := bytes.NewBuffer((*bufPtr)[:0])
				notification.UserID = pgtype.UUID{} // Remove the userID from the notification
				if err := json.NewEncoder(buf).Encode(notification); err != nil {
					slog.Error("Notifications : failed to marshal notification",
						"error", err,
						"user_id", services.UUIDToString(userID))
					continue
				}
				s.PutNotification(notification)

				// Send notification
				w.WriteString(fmt.Sprintf("data: %s\n\n", buf.Bytes()))
				if err := w.Flush(); err != nil {
					slog.Warn("Notifications : Client disconnected (flush error)",
						"error", err,
						"user_id", services.UUIDToString(userID))
					return
				}

			case <-closeChan:
				slog.Debug("Notifications : Received close signal", "user_id", services.UUIDToString(userID))
				return
			case <-connectionCheckTicker.C:
				// Check if connection is still valid
				if currentSessionId, err := s.hasActiveConnection(context.Background(), userID, cache); (err == nil && sessionId != currentSessionId) || (err != nil) {
					slog.Debug("Notifications : Connection status invalid or expired", "user_id", services.UUIDToString(userID))
					return
				}
			}
		}
	})
	return nil
}

// RefreshConnectionTTL refreshes the connection TTL for a user
func (s *NotificationService) RefreshConnectionTTL(ctx context.Context, userID pgtype.UUID, cache *services.Cache) error {
	key := fmt.Sprintf("notifications:is_connected:%s", services.UUIDToString(userID))
	return cache.Db.Expire(ctx, key, 15*time.Minute).Err()
}

// CloseConnection forcefully closes a user's connection
func (s *NotificationService) CloseConnection(c *fiber.Ctx, cache *services.Cache) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown user",
		})
	}
	sessionId, err := middleware.GetSessionId(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown session",
		})
	}

	s.registry.mu.RLock()
	closeChan, exists := s.registry.closeChans[sessionId]
	s.registry.mu.RUnlock()

	if exists {
		// Signal the SSE handler to close
		close(closeChan)
	}
	key := fmt.Sprintf("notifications:is_connected:%s", services.UUIDToString(userID))
	return cache.Db.Del(c.Context(), key).Err()
}
