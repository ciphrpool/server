package duels

import (
	"backend/lib/services"
	"context"
	"errors"
	"log/slog"
	"sync"
)

var (
	ErrSupervisorStarted = errors.New("supervisor is already started")
	ErrInitFailed        = errors.New("failed to initialize components")
	ErrNilCache          = errors.New("cache cannot be nil")
)

type DuelSupervisor struct {
	subscriber  *DuelSubscriber
	worker_pool *WorkerPool
	is_running  bool
	mu          sync.RWMutex
}

// NewDuelSupervisor creates a new supervisor
func NewDuelSupervisor(worker_size int) (*DuelSupervisor, error) {
	if worker_size <= 0 {
		return nil, errors.New("worker size must be positive")
	}

	// Create worker pool
	worker_pool := NewWorkerPool(worker_size)
	if worker_pool == nil {
		return nil, errors.New("failed to create worker pool")
	}

	// Create subscriber
	subscriber, err := NewDuelSubscriber(worker_pool)
	if err != nil {
		return nil, err
	}

	return &DuelSupervisor{
		subscriber:  subscriber,
		worker_pool: worker_pool,
		is_running:  false,
	}, nil
}

// Start begins the supervision of the subscriber and worker pool
func (s *DuelSupervisor) Start(ctx context.Context, cache *services.Cache, db *services.Database) error {
	if cache == nil {
		return ErrNilCache
	}
	if cache.Db == nil {
		return ErrNilCache
	}
	s.mu.Lock()
	if s.is_running {
		s.mu.Unlock()
		return ErrSupervisorStarted
	}
	s.mu.Unlock()

	// Start components with proper error handling
	errCh := make(chan error, 2) // Buffer for both components

	// Start worker pool
	go func() {
		s.worker_pool.Start(ctx, cache, db)
		errCh <- nil // Worker pool doesn't return error
	}()

	// Start subscriber
	go func() {
		if err := s.subscriber.Subscribe(ctx, cache); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	// Wait for both components to start
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			s.Stop(context.Background())
			return err
		}
	}

	s.mu.Lock()
	s.is_running = true
	s.mu.Unlock()

	// Monitor context cancellation
	go func() {
		<-ctx.Done()
		if err := s.Stop(context.Background()); err != nil {
			s.HandleError(err)
		}
	}()

	return nil
}

// Stop gracefully shuts down all components
func (s *DuelSupervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.is_running {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	// Create error channel to collect shutdown errors
	errCh := make(chan error, 2)

	// Stop subscriber
	go func() {
		errCh <- s.subscriber.UnSubscribe(ctx)
	}()

	// Stop worker pool
	go func() {
		s.worker_pool.Stop()
		errCh <- nil // Worker pool stop doesn't return error
	}()

	// Collect errors from both operations
	var shutdown_err error
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			shutdown_err = err
			slog.Error("component shutdown failed", "error", err)
		}
	}

	s.mu.Lock()
	s.is_running = false
	s.mu.Unlock()

	return shutdown_err
}

// HandleError processes errors from workers and subscriber
func (s *DuelSupervisor) HandleError(err error) {
	if err == nil {
		return
	}

	// Log the error with context
	slog.Error("duel processing error",
		"error", err,
		"component", "supervisor")

	// Add specific error handling based on error types
	switch {
	case errors.Is(err, context.Canceled):
		slog.Info("supervisor shutdown due to context cancellation")
	case errors.Is(err, ErrPoolNotStarted):
		slog.Error("worker pool failed to start")
	default:
		slog.Error("unexpected error in duel processing", "error", err)
	}
}
