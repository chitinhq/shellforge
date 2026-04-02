// Package scheduler provides a priority-aware inference queue with
// concurrency control. This prevents local model overload by limiting
// parallel inference slots. Requests are dispatched in descending
// Priority order; within the same priority they are served FIFO.
package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
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

// waitEntry holds a pending Submit() call in the priority heap.
type waitEntry struct {
	req   InferenceRequest
	grant chan struct{} // closed when a slot is granted to this entry
	seq   int64        // insertion order for FIFO tie-breaking within equal priority
	index int          // position in heap; -1 once removed or dispatched
}

// requestHeap is a max-heap over *waitEntry ordered by Priority (descending),
// then by seq (ascending, i.e. FIFO within equal priority).
type requestHeap []*waitEntry

func (h requestHeap) Len() int { return len(h) }

func (h requestHeap) Less(i, j int) bool {
	if h[i].req.Priority != h[j].req.Priority {
		return h[i].req.Priority > h[j].req.Priority
	}
	return h[i].seq < h[j].seq
}

func (h requestHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *requestHeap) Push(x any) {
	e := x.(*waitEntry)
	e.index = len(*h)
	*h = append(*h, e)
}

func (h *requestHeap) Pop() any {
	old := *h
	n := len(old)
	e := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	e.index = -1
	return e
}

// InferenceQueue manages concurrent access to local model inference.
// Requests wait in a max-priority heap and are dispatched in Priority
// order (higher values first). Within the same priority, requests are
// served FIFO.
type InferenceQueue struct {
	mu          sync.Mutex
	waiting     requestHeap
	maxParallel int
	running     int
	maxDepth    int
	pending     int64 // atomic: waiting + running
	seq         int64 // monotonic insertion counter
}

// NewInferenceQueue creates a queue that dispatches up to maxParallel
// concurrent inference calls and rejects new submissions when maxDepth
// requests are already pending (waiting + running).
func NewInferenceQueue(maxParallel, maxDepth int) *InferenceQueue {
	q := &InferenceQueue{
		maxParallel: maxParallel,
		maxDepth:    maxDepth,
	}
	heap.Init(&q.waiting)
	return q
}

// Submit enqueues req and blocks until a slot is granted or ctx is
// cancelled. On success it returns a release function that the caller
// MUST invoke once inference is complete to free the slot.
func (q *InferenceQueue) Submit(ctx context.Context, req InferenceRequest) (func(), error) {
	if int(atomic.LoadInt64(&q.pending)) >= q.maxDepth {
		return nil, fmt.Errorf("inference queue full (%d pending)", q.maxDepth)
	}
	atomic.AddInt64(&q.pending, 1)

	entry := &waitEntry{
		req:   req,
		grant: make(chan struct{}),
		seq:   atomic.AddInt64(&q.seq, 1),
	}

	q.mu.Lock()
	heap.Push(&q.waiting, entry)
	q.tryDispatch()
	q.mu.Unlock()

	select {
	case <-entry.grant:
		return func() {
			atomic.AddInt64(&q.pending, -1)
			q.mu.Lock()
			q.running--
			q.tryDispatch()
			q.mu.Unlock()
		}, nil

	case <-ctx.Done():
		q.mu.Lock()
		if entry.index >= 0 {
			// Still in heap — remove cleanly before it is dispatched.
			heap.Remove(&q.waiting, entry.index)
			q.mu.Unlock()
			atomic.AddInt64(&q.pending, -1)
			return nil, ctx.Err()
		}
		// index == -1: tryDispatch already popped this entry and incremented
		// q.running. Release the slot immediately since we won't use it.
		q.running--
		q.tryDispatch()
		q.mu.Unlock()
		atomic.AddInt64(&q.pending, -1)
		return nil, ctx.Err()
	}
}

// tryDispatch grants slots to the highest-priority waiting entries until
// maxParallel is reached or the heap is empty. Must be called with q.mu held.
func (q *InferenceQueue) tryDispatch() {
	for q.running < q.maxParallel && len(q.waiting) > 0 {
		entry := heap.Pop(&q.waiting).(*waitEntry)
		q.running++
		close(entry.grant)
	}
}

// Pending returns the current number of pending inference requests
// (both waiting for a slot and currently running).
func (q *InferenceQueue) Pending() int64 {
	return atomic.LoadInt64(&q.pending)
}

// MaxParallel returns the concurrency limit for this queue.
func (q *InferenceQueue) MaxParallel() int {
	return q.maxParallel
}
