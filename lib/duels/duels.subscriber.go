package duels

import (
	"backend/lib/services"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	ErrNilWorkerPool = errors.New("worker pool cannot be nil")
	ErrEmptyChannel  = errors.New("channel name cannot be empty")
)

var (
	DuelResultChannel = "duel:result:*"
)

type DuelSubscriber struct {
	worker_pool *WorkerPool
	channel     string
	pubsub      *redis.PubSub
	mu          sync.Mutex
	is_active   bool
}

// NewDuelSubscriber creates a new subscriber connected to Redis
func NewDuelSubscriber(worker_pool *WorkerPool) (*DuelSubscriber, error) {
	if worker_pool == nil {
		return nil, ErrNilWorkerPool
	}

	return &DuelSubscriber{
		worker_pool: worker_pool,
		channel:     DuelResultChannel,
		is_active:   false,
	}, nil
}

// Subscribe starts listening for duel results
func (s *DuelSubscriber) Subscribe(ctx context.Context, cache *services.Cache) error {
	s.mu.Lock()
	if s.is_active {
		s.mu.Unlock()
		return errors.New("subscriber is already active")
	}

	// Initialize PubSub
	slog.Debug("Subscribing to the duel result channel", "channel", s.channel)
	s.pubsub = cache.Db.PSubscribe(ctx, s.channel)
	s.is_active = true
	s.mu.Unlock()

	// Start message processing in a separate goroutine
	go func() {
		ch := s.pubsub.Channel()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					slog.Info("pubsub channel closed")
					return
				}

				slog.Debug("Received a result from the channel")
				session_id := strings.TrimPrefix(msg.Channel, "duel:result:")

				// Process the message
				if err := s.processMessage(ctx, msg.Payload, session_id); err != nil {
					slog.Error("failed to process message",
						"error", err,
						"channel", msg.Channel)
					continue
				}

			case <-ctx.Done():
				slog.Info("context cancelled, stopping subscriber")
				s.UnSubscribe(context.Background())
				return
			}
		}
	}()

	return nil
}

// UnSubscribe stops listening for messages
func (s *DuelSubscriber) UnSubscribe(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.is_active {
		return nil
	}

	slog.Debug("Unsubscribing to the duel result channel", "channel", s.channel)
	if err := s.pubsub.PUnsubscribe(ctx, s.channel); err != nil {
		return err
	}

	if err := s.pubsub.Close(); err != nil {
		return err
	}

	s.is_active = false
	return nil
}

func (s *DuelSubscriber) processMessage(ctx context.Context, message string, session_id string) error {
	if message == "" {
		return errors.New("empty message received")
	}

	var result DuelResult
	if err := json.Unmarshal([]byte(message), &result); err != nil {
		return err
	}
	result.SessionID = session_id

	return s.worker_pool.SubmitResult(&result)
}
