package duels

import (
	"backend/lib/services"
	"context"
	"errors"
	"log/slog"
	"sync"

	basepool "github.com/ciphrpool/base-pool/gen"
)

var (
	ErrNilResult = errors.New("cannot process nil result")
)

type DuelWorker struct {
	work_chan chan *DuelResult
	pool      *sync.Pool
	is_active bool
	mu        sync.RWMutex
}

// NewDuelWorker creates a new worker with its own work channel
func NewDuelWorker() *DuelWorker {
	worker := &DuelWorker{
		work_chan: make(chan *DuelResult, 100), // Buffer for better throughput
		is_active: false,
	}

	// Initialize sync.Pool for DuelResult processing
	worker.pool = &sync.Pool{
		New: func() interface{} {
			return new(DuelResult)
		},
	}

	return worker
}

// Process handles a single duel result
func (w *DuelWorker) Process(ctx context.Context, result *DuelResult, cache *services.Cache, db *services.Database) error {
	if cache == nil {
		return ErrNilCache
	}
	if cache.Db == nil {
		return ErrNilCache
	}
	if result == nil {
		return ErrNilResult
	}

	// Get a result object from the pool
	pooled_result := w.pool.Get().(*DuelResult)
	defer w.pool.Put(pooled_result)

	// Copy the result data to our pooled object
	*pooled_result = *result

	slog.Debug("Processing Duel Result", "result", pooled_result)
	switch pooled_result.SessionData.DuelType {
	case basepool.DuelTypeFriendly:
		if err := FriendlyDuelResultProcessor(ctx, pooled_result, cache, db); err != nil {
			return err
		}
	case basepool.DuelTypeRanked:
		if err := RankedDuelResultProcessor(ctx, pooled_result, cache, db); err != nil {
			return err
		}
	case basepool.DuelTypeTournament:
		if err := TournamentDuelResultProcessor(ctx, pooled_result, cache, db); err != nil {
			return err
		}
	}

	return nil
}

// Start begins the worker's processing loop
func (w *DuelWorker) Start(ctx context.Context) {
	w.mu.Lock()
	if w.is_active {
		w.mu.Unlock()
		return
	}
	w.is_active = true
	w.mu.Unlock()
}

// Stop gracefully shuts down the worker
func (w *DuelWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.is_active {
		return
	}

	close(w.work_chan)
	w.is_active = false
}
