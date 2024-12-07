package notifications

import (
	"backend/lib/services"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

// Common errors
var (
	ErrInvalidConfig    = errors.New("invalid configuration")
	ErrWorkerTimeout    = errors.New("worker timeout")
	ErrDeliveryFailed   = errors.New("notification delivery failed")
	ErrConnectionClosed = errors.New("connection closed")
)

// NotificationType represents different types of notifications
type NotificationType string

const (
	TypeRedirect NotificationType = "redirect"
	TypeMessage  NotificationType = "message"
	TypePing     NotificationType = "ping"
	TypeAlert    NotificationType = "alert"
)

type NotificationPriority int

const (
	PriorityLow    NotificationPriority = 1
	PriorityMedium NotificationPriority = 2
	PriorityHigh   NotificationPriority = 3
)

// Notification represents a single notification message
type Notification struct {
	ID        string               `json:"id"`
	Type      NotificationType     `json:"type"`
	UserID    pgtype.UUID          `json:"user_id"`
	Content   fiber.Map            `json:"content"`
	CreatedAt time.Time            `json:"created_at"`
	Priority  NotificationPriority `json:"priority"`
	Metadata  fiber.Map            `json:"metadata,omitempty"`
}

func (n *Notification) Reset() {
	n.ID = ""
	n.Type = ""
	n.UserID = pgtype.UUID{}
	n.Content = nil
	n.CreatedAt = time.Time{}
	n.Priority = 0
	if n.Metadata != nil {
		for k := range n.Metadata {
			delete(n.Metadata, k)
		}
	}
}

// Config holds the configuration for the notification system
type Config struct {
	WorkerCount       int
	WorkerQueueSize   int
	RetryAttempts     int
	RetryDelay        time.Duration
	ShutdownTimeout   time.Duration
	HeartbeatInterval time.Duration
	InitialPoolSize   int
	SessionTimeout    time.Duration
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {

	if c.WorkerCount < 1 {
		return fmt.Errorf("%w: worker count must be greater than 0", ErrInvalidConfig)
	}
	return nil
}

// clientRegistry manages active SSE connections
type clientRegistry struct {
	mu         sync.RWMutex
	clients    map[pgtype.UUID]chan *Notification
	subConns   map[pgtype.UUID]*redis.PubSub
	closeChans map[pgtype.UUID]chan struct{}
}

// NotificationService represents the notification service
type NotificationService struct {
	config   *Config
	workers  chan struct{}
	jobs     chan *Notification
	shutdown chan struct{}
	wg       sync.WaitGroup
	registry *clientRegistry
	pool     sync.Pool // Pool for notification objects
	bufPool  sync.Pool // Pool for JSON encoding buffers
}

// NewNotificationService creates a new notification service
func NewNotificationService(cfg *Config) (*NotificationService, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	s := &NotificationService{
		config:   cfg,
		workers:  make(chan struct{}, cfg.WorkerCount),
		jobs:     make(chan *Notification, cfg.WorkerQueueSize),
		shutdown: make(chan struct{}),
		registry: &clientRegistry{
			clients:    make(map[pgtype.UUID]chan *Notification),
			subConns:   make(map[pgtype.UUID]*redis.PubSub),
			closeChans: make(map[pgtype.UUID]chan struct{}),
		},
	}
	// Initialize notification pool
	s.pool.New = func() interface{} {
		return &Notification{}
	}
	// Initialize buffer pool
	s.bufPool.New = func() interface{} {
		b := make([]byte, 0, 1024) // Pre-allocate with reasonable size
		return &b                  // Return pointer to byte slice
	}

	// Pre-warm the pool
	if cfg.InitialPoolSize > 0 {
		for i := 0; i < cfg.InitialPoolSize; i++ {
			s.pool.Put(&Notification{})
		}
	}

	return s, nil
}

// GetNotification gets a notification from the pool
func (s *NotificationService) GetNotification() *Notification {
	return s.pool.Get().(*Notification)
}

// PutNotification returns a notification to the pool
func (s *NotificationService) PutNotification(n *Notification) {
	n.Reset()
	s.pool.Put(n)
}

// Send queues a notification for processing
func (s *NotificationService) Send(
	ctx context.Context,
	t NotificationType,
	priority NotificationPriority,
	user_id pgtype.UUID,
	content fiber.Map,
	metadata fiber.Map,
) error {
	// Create a pooled copy of the notification
	pooledNotification := s.GetNotification()
	pooledNotification.CreatedAt = time.Now()
	pooledNotification.ID = uuid.New().String()
	pooledNotification.UserID = user_id
	pooledNotification.Type = t
	pooledNotification.Priority = priority
	pooledNotification.Content = content
	pooledNotification.Metadata = metadata

	select {
	case s.jobs <- pooledNotification:
		return nil
	case <-ctx.Done():
		s.PutNotification(pooledNotification) // Return to pool if not sent
		return ctx.Err()
	}
}

// processNotification handles the delivery of a single notification
func (s *NotificationService) processNotification(ctx context.Context, n *Notification, cache *services.Cache) error {
	defer s.PutNotification(n) // Return notification to pool after processing

	if s.hasActiveConnection(ctx, n.UserID, cache) {
		return s.deliverNotification(ctx, n, cache)
	}
	if n.Type != TypeRedirect {
		return s.storeNotification(ctx, n, cache)
	} else {
		// the notification will not be stored and will be discarded immediatly
		return nil
	}
}

// deliverNotification handles delivering a notification via Redis pub/sub
func (s *NotificationService) deliverNotification(ctx context.Context, n *Notification, cache *services.Cache) error {
	channel := fmt.Sprintf("notifications:channel:user:%s", services.UUIDToString(n.UserID))

	// Get buffer from pool
	bufPtr := s.bufPool.Get().(*[]byte)
	b := bytes.NewBuffer((*bufPtr)[:0])
	defer s.bufPool.Put(bufPtr)

	// Encode directly into our buffer
	if err := json.NewEncoder(b).Encode(n); err != nil {
		return err
	}

	return cache.Db.Publish(ctx, channel, b.Bytes()).Err()
}

// Start initializes the worker pool and starts processing notifications
func (s *NotificationService) Start(ctx context.Context, cache *services.Cache) error {
	slog.Info("Notifications : starting notification service")

	// Initialize worker pool
	for i := 0; i < s.config.WorkerCount; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i, cache)
	}

	return nil
}

// registerClient adds a new SSE client connection
func (s *NotificationService) registerClient(ctx context.Context, userID pgtype.UUID, notifications chan *Notification, cache *services.Cache) error {
	s.registry.mu.Lock()
	defer s.registry.mu.Unlock()

	// Check if client already exists
	if _, exists := s.registry.clients[userID]; exists {
		return fmt.Errorf("client already registered: %s", services.UUIDToString(userID))
	}

	// Subscribe to Redis channel
	channel := fmt.Sprintf("notifications:channel:user:%s", services.UUIDToString(userID))
	pubsub := cache.Db.Subscribe(ctx, channel)
	closeChan := make(chan struct{})

	// Verify subscription
	_, err := pubsub.Receive(ctx)
	if err != nil {
		pubsub.Close()
		return fmt.Errorf("failed to subscribe to Redis channel: %w", err)
	}

	// Store client channel and subscription
	s.registry.clients[userID] = notifications
	s.registry.subConns[userID] = pubsub
	s.registry.closeChans[userID] = closeChan

	// Start message relay goroutine
	go s.relayMessages(ctx, userID, pubsub.Channel(), notifications, cache)

	// Set connection status in Redis
	err = cache.Db.Set(ctx, fmt.Sprintf("notifications:is_connected:%s", services.UUIDToString(userID)), true, 15*time.Minute).Err()
	if err != nil {
		slog.Error("Notifications : failed to set connection status", "error", err, "user_id", services.UUIDToString(userID))
	}

	slog.Info("Notifications : client registered to a notification session", "user_id", services.UUIDToString(userID))
	return nil
}

// unregisterClient removes a client connection
func (s *NotificationService) unregisterClient(ctx context.Context, userID pgtype.UUID, cache *services.Cache) {
	s.registry.mu.Lock()
	defer s.registry.mu.Unlock()

	// Close Redis subscription if exists
	if pubsub, exists := s.registry.subConns[userID]; exists {
		if err := pubsub.Close(); err != nil {
			slog.Error("Notifications : failed to close pubsub connection",
				"error", err,
				"user_id", services.UUIDToString(userID))
		}
		delete(s.registry.subConns, userID)
	}

	// Remove client channel
	delete(s.registry.clients, userID)
	delete(s.registry.closeChans, userID)

	// Remove connection status from Redis
	if s.hasActiveConnection(ctx, userID, cache) {
		err := cache.Db.Del(ctx, fmt.Sprintf("notifications:is_connected:%s", services.UUIDToString(userID))).Err()
		if err != nil {
			slog.Error("Notifications : failed to remove connection status",
				"error", err,
				"user_id", services.UUIDToString(userID))
		}
	}

	slog.Info("Notifications : client has been unregistered from its notification session", "user_id", services.UUIDToString(userID))
}

// relayMessages handles message relay from Redis to SSE client
func (s *NotificationService) relayMessages(ctx context.Context, userID pgtype.UUID, redisMessages <-chan *redis.Message, notifications chan<- *Notification, cache *services.Cache) {
	for {
		select {
		case msg, ok := <-redisMessages:
			if !ok {
				return
			}
			notification := s.GetNotification()

			if err := json.Unmarshal([]byte(msg.Payload), notification); err != nil {
				slog.Error("Notifications : failed to unmarshal notification",
					"error", err,
					"user_id", userID)
				s.PutNotification(notification)
				continue
			}
			// Check if client still exists with lock
			s.registry.mu.RLock()
			_, exists := s.registry.clients[userID]
			s.registry.mu.RUnlock()

			if !exists {
				s.PutNotification(notification)
				return
			}

			select {
			case notifications <- notification:
			case <-ctx.Done():
				s.PutNotification(notification)
				return
			case <-s.shutdown:
				s.PutNotification(notification)
				return
			default:
				slog.Warn("Notifications : client channel full, storing notification",
					"user_id", userID)
				if err := s.storeNotification(ctx, notification, cache); err != nil {
					slog.Error("Notifications : failed to store notification",
						"error", err,
						"user_id", userID)
				}
				s.PutNotification(notification)
			}

		case <-ctx.Done():
			return
		case <-s.shutdown:
			return
		}
	}
}

// deliverStoredNotifications sends previously stored notifications to the client
func (s *NotificationService) deliverStoredNotifications(ctx context.Context, userID pgtype.UUID, cache *services.Cache) error {
	key := fmt.Sprintf("notifications:buffer:%s", services.UUIDToString(userID))

	for {
		result, err := cache.Db.LPop(ctx, key).Result()
		if err == redis.Nil {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to get stored notification: %w", err)
		}

		notification := s.GetNotification()

		if err := json.Unmarshal([]byte(result), notification); err != nil {
			s.PutNotification(notification)
			continue
		}

		s.registry.mu.RLock()
		notifications, exists := s.registry.clients[userID]
		s.registry.mu.RUnlock()

		if !exists {
			s.PutNotification(notification)
			return fmt.Errorf("client not found: %s", services.UUIDToString(userID))
		}

		select {
		case notifications <- notification:
		case <-ctx.Done():
			s.PutNotification(notification)
			return ctx.Err()
		}
	}

	return nil
}

// worker processes notifications from the job queue
func (s *NotificationService) worker(ctx context.Context, id int, cache *services.Cache) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case notification := <-s.jobs:
			if err := s.processNotification(ctx, notification, cache); err != nil {
				slog.Error("Notifications : failed to process notification",
					"error", err,
					"notification_id", notification.ID)

				// Handle retry logic
				if err := s.handleRetry(ctx, notification, cache); err != nil {
					slog.Error("Notifications : retry failed", "error", err)
				}
			}
		}
	}
}

// Shutdown gracefully shuts down the service
func (s *NotificationService) Shutdown(ctx context.Context) error {
	slog.Info("Notifications : shutting down notification service")

	close(s.shutdown)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Helper methods

func (s *NotificationService) hasActiveConnection(ctx context.Context, userID pgtype.UUID, cache *services.Cache) bool {
	count, err := cache.Db.Exists(ctx, fmt.Sprintf("notifications:is_connected:%s", services.UUIDToString(userID))).Result()
	if err != nil {
		return false
	}
	return count > 0
}

func (s *NotificationService) storeNotification(ctx context.Context, n *Notification, cache *services.Cache) error {
	slog.Debug("Storing received notification", "notification", n)
	key := fmt.Sprintf("notifications:buffer:%s", services.UUIDToString(n.UserID))
	data, err := json.Marshal(n)
	if err != nil {
		return err
	}

	return cache.Db.RPush(ctx, key, data).Err()
}

func (s *NotificationService) handleRetry(ctx context.Context, n *Notification, cache *services.Cache) error {
	for i := 0; i < s.config.RetryAttempts; i++ {
		time.Sleep(s.config.RetryDelay)

		if err := s.processNotification(ctx, n, cache); err == nil {
			return nil
		}
	}

	return ErrDeliveryFailed
}
