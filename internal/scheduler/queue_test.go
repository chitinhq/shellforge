package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func req(priority Priority) InferenceRequest {
	return InferenceRequest{Priority: priority}
}

// TestPriorityOrdering verifies that a higher-priority request is granted
// before a lower-priority one when both are waiting for the same slot.
func TestPriorityOrdering(t *testing.T) {
	q := NewInferenceQueue(1, 10)

	// Hold the single slot.
	release0, err := q.Submit(context.Background(), req(PriorityWorker))
	if err != nil {
		t.Fatalf("initial submit: %v", err)
	}

	// Enqueue a low-priority and a high-priority request while slot is held.
	var order []string
	var wg sync.WaitGroup
	mu := sync.Mutex{}

	wg.Add(2)
	// Submit low-priority first so it enters the heap before high-priority.
	go func() {
		defer wg.Done()
		release, err := q.Submit(context.Background(), req(PriorityWorker))
		if err != nil {
			t.Errorf("low-priority submit: %v", err)
			return
		}
		mu.Lock()
		order = append(order, "low")
		mu.Unlock()
		release()
	}()
	// Small delay so the low-priority goroutine enters the heap first.
	time.Sleep(5 * time.Millisecond)
	go func() {
		defer wg.Done()
		release, err := q.Submit(context.Background(), req(PriorityEvaluator))
		if err != nil {
			t.Errorf("high-priority submit: %v", err)
			return
		}
		mu.Lock()
		order = append(order, "high")
		mu.Unlock()
		release()
	}()
	time.Sleep(5 * time.Millisecond)

	// Freeing the held slot should dispatch the Evaluator (higher priority) first.
	release0()
	wg.Wait()

	if len(order) != 2 {
		t.Fatalf("expected 2 completions, got %d", len(order))
	}
	if order[0] != "high" {
		t.Errorf("expected high-priority first, got order=%v", order)
	}
}

// TestFIFOWithinSamePriority verifies that equal-priority requests are
// served in submission order.
func TestFIFOWithinSamePriority(t *testing.T) {
	q := NewInferenceQueue(1, 10)

	// Hold the single slot.
	hold, _ := q.Submit(context.Background(), req(PriorityWorker))

	var order []int
	var wg sync.WaitGroup
	mu := sync.Mutex{}

	for i := 0; i < 3; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := q.Submit(context.Background(), req(PriorityWorker))
			if err != nil {
				t.Errorf("submit %d: %v", i, err)
				return
			}
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
			release()
		}()
		time.Sleep(2 * time.Millisecond) // ensure ordered insertion
	}
	time.Sleep(10 * time.Millisecond)

	hold()
	wg.Wait()

	for i, v := range order {
		if v != i {
			t.Errorf("expected FIFO order [0 1 2], got %v", order)
			break
		}
	}
}

// TestConcurrencyLimit verifies that at most maxParallel requests run at once.
func TestConcurrencyLimit(t *testing.T) {
	const maxP = 2
	q := NewInferenceQueue(maxP, 20)

	var running int64
	var peak int64
	var wg sync.WaitGroup

	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := q.Submit(context.Background(), req(PriorityWorker))
			if err != nil {
				t.Errorf("submit: %v", err)
				return
			}
			cur := atomic.AddInt64(&running, 1)
			for {
				old := atomic.LoadInt64(&peak)
				if cur <= old || atomic.CompareAndSwapInt64(&peak, old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt64(&running, -1)
			release()
		}()
	}
	wg.Wait()

	if peak > maxP {
		t.Errorf("concurrency limit violated: peak=%d, maxParallel=%d", peak, maxP)
	}
}

// TestQueueFull verifies rejection when maxDepth is reached.
func TestQueueFull(t *testing.T) {
	q := NewInferenceQueue(1, 2)

	// Hold the one slot.
	hold, _ := q.Submit(context.Background(), req(PriorityWorker))

	blocker := make(chan struct{})
	go func() {
		release, err := q.Submit(context.Background(), req(PriorityWorker))
		if err == nil {
			release()
		}
		close(blocker)
	}()
	time.Sleep(5 * time.Millisecond) // let goroutine enter heap (pending=2)

	_, err := q.Submit(context.Background(), req(PriorityWorker))
	if err == nil {
		t.Error("expected queue-full error, got nil")
	}

	hold() // release slot so the goroutine can proceed
	<-blocker
}

// TestContextCancellationWhileWaiting verifies that a cancelled context
// removes the entry from the heap and returns the context error.
func TestContextCancellationWhileWaiting(t *testing.T) {
	q := NewInferenceQueue(1, 10)

	// Hold the slot so the next Submit must wait.
	hold, _ := q.Submit(context.Background(), req(PriorityWorker))
	defer hold()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := q.Submit(ctx, req(PriorityWorker))
	if err == nil {
		t.Error("expected context error, got nil")
	}
	if q.Pending() != 1 { // only the held slot should remain
		t.Errorf("expected pending=1 after cancellation, got %d", q.Pending())
	}
}

// TestReleaseRestoresSlot verifies that calling release() makes the slot
// available for the next waiter.
func TestReleaseRestoresSlot(t *testing.T) {
	q := NewInferenceQueue(1, 10)

	release, err := q.Submit(context.Background(), req(PriorityWorker))
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	if q.Pending() != 1 {
		t.Errorf("expected pending=1 while slot held, got %d", q.Pending())
	}

	done := make(chan struct{})
	go func() {
		r2, err := q.Submit(context.Background(), req(PriorityWorker))
		if err != nil {
			t.Errorf("second submit: %v", err)
		} else {
			r2()
		}
		close(done)
	}()

	time.Sleep(5 * time.Millisecond)
	release() // should unblock the waiter

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Error("second Submit did not complete after release()")
	}

	if q.Pending() != 0 {
		t.Errorf("expected pending=0 after all releases, got %d", q.Pending())
	}
}

// TestMaxParallelReturnsCorrectValue is a basic smoke test for MaxParallel().
func TestMaxParallelReturnsCorrectValue(t *testing.T) {
	q := NewInferenceQueue(3, 10)
	if q.MaxParallel() != 3 {
		t.Errorf("expected MaxParallel=3, got %d", q.MaxParallel())
	}
}
