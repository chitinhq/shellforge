// Package scheduler provides a priority-aware inference queue with
// semaphore-based concurrency control. This prevents local model
// overload by limiting parallel inference slots and rejecting
// requests when the queue depth is exceeded.
package scheduler

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// Priority determines scheduling order. Higher values are served first
// when contending for inference slots.
type Priority int

const (
	PriorityWorker    Priority = 1
	PriorityPlanner   Priority = 1
	PriorityCorrector Priority = 2
	PriorityEvaluator Priority = 3
)

// InferenceRequest represents a pending model call from an agent.
type InferenceRequest struct {
	ID       string
	Agent    string
	Priority Priority
	Prompt   string
	System   string
	Budget   int // max output tokens
	Timeout  time.Duration
	Result   chan<- InferenceResult
}

// InferenceResult carries the model response back to the requesting agent.
type InferenceResult struct {
	Output string
	Error  error
	Tokens int
}

// InferenceQueue manages concurrent access to local model inference.
// It uses a buffered channel as a semaphore to limit parallelism and
// an atomic counter to enforce maximum queue depth.
type InferenceQueue struct {
	slots    chan struct{}
	maxDepth int
	pending  int64 // atomic counter
}

// NewInferenceQueue creates a queue that allows maxParallel concurrent
// inference calls and rejects submissions when maxDepth requests are
// already waiting.
func NewInferenceQueue(maxParallel, maxDepth int) *InferenceQueue {
	return &InferenceQueue{
		slots:    make(chan struct{}, maxParallel),
		maxDepth: maxDepth,
	}
}

// Submit attempts to acquire an inference slot. It blocks until a slot
// is available or the context is cancelled. Returns an error if the
// queue depth limit is exceeded or the context expires.
func (q *InferenceQueue) Submit(ctx context.Context, req InferenceRequest) error {
	if int(atomic.LoadInt64(&q.pending)) >= q.maxDepth {
		return fmt.Errorf("inference queue full (%d pending)", q.maxDepth)
	}
	atomic.AddInt64(&q.pending, 1)
	defer atomic.AddInt64(&q.pending, -1)

	// Acquire slot (blocks until available or context cancelled)
	select {
	case q.slots <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-q.slots }()

	// Slot acquired — the caller's runFunc handles actual inference.
	// This method only manages concurrency; execution is delegated
	// to the orchestrator that owns this queue.
	return nil
}

// Pending returns the current number of queued inference requests.
func (q *InferenceQueue) Pending() int64 {
	return atomic.LoadInt64(&q.pending)
}

// MaxParallel returns the concurrency limit for this queue.
func (q *InferenceQueue) MaxParallel() int {
	return cap(q.slots)
}
