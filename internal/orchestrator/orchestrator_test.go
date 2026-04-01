package orchestrator

import (
	"testing"
)

func TestNewOrchestrator_MinParallel(t *testing.T) {
	// maxParallel < 1 should default to 1
	o := NewOrchestrator(nil, nil, 0)
	if o.maxParallel != 1 {
		t.Errorf("expected maxParallel=1, got %d", o.maxParallel)
	}
}

func TestNewOrchestrator_SetsFields(t *testing.T) {
	o := NewOrchestrator(nil, nil, 4)
	if o.maxParallel != 4 {
		t.Errorf("expected maxParallel=4, got %d", o.maxParallel)
	}
	if cap(o.slots) != 4 {
		t.Errorf("expected slots capacity=4, got %d", cap(o.slots))
	}
	// All slots should be available initially
	if len(o.slots) != 4 {
		t.Errorf("expected 4 available slots, got %d", len(o.slots))
	}
}

func TestNewOrchestrator_NegativeParallel(t *testing.T) {
	o := NewOrchestrator(nil, nil, -5)
	if o.maxParallel != 1 {
		t.Errorf("expected maxParallel=1 for negative input, got %d", o.maxParallel)
	}
}
