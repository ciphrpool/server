package duels

import (
	"backend/lib/services"
	"context"
	"errors"
	"log/slog"
	"sync"
)

var (
	ErrPoolNotStarted = errors.New("worker pool not started")
	ErrPoolClosed     = errors.New("worker pool is closed")
)

type WorkerPool struct {
	workers     []*DuelWorker
	result_chan chan *DuelResult
	worker_size int
	started     bool
	mu          sync.RWMutex
}

// NewWorkerPool creates a new pool with the specified number of workers
func NewWorkerPool(worker_size int) *WorkerPool {
	if worker_size <= 0 {
		worker_size = 1
	}

	return &WorkerPool{
		workers:     make([]*DuelWorker, worker_size),
		result_chan: make(chan *DuelResult, worker_size*2), // Buffer size is 2x worker count for better throughput
		worker_size: worker_size,
		started:     false,
	}
}

// SubmitResult sends a duel result to be processed by the worker pool
func (p *WorkerPool) SubmitResult(result *DuelResult) error {
	if result == nil {
		return errors.New("cannot submit nil result")
	}

	slog.Debug("Submit the result of the duel", "SessionID", result.SessionID)
	p.mu.RLock()
	if !p.started {
		p.mu.RUnlock()
		return ErrPoolNotStarted
	}
	p.mu.RUnlock()

	select {
	case p.result_chan <- result:
		return nil
	default:
		return errors.New("worker pool queue is full")
	}
}

// Start initializes and starts all workers in the pool
func (p *WorkerPool) Start(ctx context.Context, cache *services.Cache, db *services.Database) {

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return
	}

	// Initialize workers
	slog.Debug("Starting the Duels Worker Pool", "worker_size", p.worker_size)
	for i := 0; i < p.worker_size; i++ {
		worker := NewDuelWorker()
		p.workers[i] = worker
		go func(w *DuelWorker) {
			for {
				select {
				case result, ok := <-p.result_chan:
					if !ok {
						return
					}
					if err := w.Process(ctx, result, cache, db); err != nil {
						continue // TODO : handle error
					}
				case <-ctx.Done():
					return
				}
			}
		}(worker)
	}

	p.started = true
}

// Stop gracefully shuts down all workers
func (p *WorkerPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	// Close result channel to signal workers to stop
	close(p.result_chan)

	// Stop all workers
	slog.Debug("Stopping the Duels Worker Pool")
	for _, worker := range p.workers {
		worker.Stop()
	}

	p.started = false
}
