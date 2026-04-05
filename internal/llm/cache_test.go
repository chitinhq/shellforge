package llm

import (
	"errors"
	"testing"
)

// mockProvider returns pre-configured responses in order.
type mockProvider struct {
	responses []*Response
	callCount int
	name      string
	err       error // if set, Chat returns this error
}

func (m *mockProvider) Chat(messages []Message, tools []ToolDef) (*Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	idx := m.callCount
	m.callCount++
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return &Response{}, nil
}

func (m *mockProvider) Name() string {
	return m.name
}

func TestCacheHitIncrementsHitsAndTotalHit(t *testing.T) {
	mp := &mockProvider{
		responses: []*Response{
			{CacheHit: 500},
		},
	}
	ct := NewCacheTracker(mp)

	_, err := ct.Chat([]Message{{Role: "user", Content: "hello"}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := ct.Stats()
	if s.Hits != 1 {
		t.Errorf("Hits = %d, want 1", s.Hits)
	}
	if s.TotalHit != 500 {
		t.Errorf("TotalHit = %d, want 500", s.TotalHit)
	}
	if s.SavedTokens != 500 {
		t.Errorf("SavedTokens = %d, want 500", s.SavedTokens)
	}
}

func TestCacheMissIncrementsOnNoCacheActivity(t *testing.T) {
	mp := &mockProvider{
		responses: []*Response{
			{CacheCreated: 0, CacheHit: 0},
		},
	}
	ct := NewCacheTracker(mp)

	_, err := ct.Chat([]Message{{Role: "user", Content: "hello"}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := ct.Stats()
	if s.Misses != 1 {
		t.Errorf("Misses = %d, want 1", s.Misses)
	}
	if s.Hits != 0 {
		t.Errorf("Hits = %d, want 0", s.Hits)
	}
}

func TestCacheBreakDetected(t *testing.T) {
	// Two calls with the same messages: first creates cache, second should hit
	// but instead creates again — that's a break.
	mp := &mockProvider{
		responses: []*Response{
			{CacheCreated: 300}, // first call: normal creation
			{CacheCreated: 300}, // second call: same fingerprint, creation = break
		},
	}
	ct := NewCacheTracker(mp)

	msgs := []Message{{Role: "user", Content: "same content"}}

	_, _ = ct.Chat(msgs, nil)
	_, _ = ct.Chat(msgs, nil)

	s := ct.Stats()
	if s.Breaks != 1 {
		t.Errorf("Breaks = %d, want 1", s.Breaks)
	}
	if s.TotalCreated != 600 {
		t.Errorf("TotalCreated = %d, want 600", s.TotalCreated)
	}
}

func TestNormalCreationOnNewContent(t *testing.T) {
	// Two calls with different messages: both create cache, no break.
	mp := &mockProvider{
		responses: []*Response{
			{CacheCreated: 200},
			{CacheCreated: 250},
		},
	}
	ct := NewCacheTracker(mp)

	_, _ = ct.Chat([]Message{{Role: "user", Content: "first request"}}, nil)
	_, _ = ct.Chat([]Message{{Role: "user", Content: "second request"}}, nil)

	s := ct.Stats()
	if s.Breaks != 0 {
		t.Errorf("Breaks = %d, want 0", s.Breaks)
	}
	if s.TotalCreated != 450 {
		t.Errorf("TotalCreated = %d, want 450", s.TotalCreated)
	}
}

func TestStatsCumulativeTotals(t *testing.T) {
	mp := &mockProvider{
		responses: []*Response{
			{CacheCreated: 100},                   // request 1: creation, new fingerprint
			{CacheHit: 80},                        // request 2: hit
			{CacheHit: 120},                       // request 3: hit
			{CacheCreated: 0, CacheHit: 0},        // request 4: miss
		},
	}
	ct := NewCacheTracker(mp)

	msgs := []Message{{Role: "user", Content: "a"}}
	for i := 0; i < 4; i++ {
		_, _ = ct.Chat(msgs, nil)
	}

	s := ct.Stats()
	if s.Requests != 4 {
		t.Errorf("Requests = %d, want 4", s.Requests)
	}
	if s.TotalCreated != 100 {
		t.Errorf("TotalCreated = %d, want 100", s.TotalCreated)
	}
	if s.TotalHit != 200 {
		t.Errorf("TotalHit = %d, want 200", s.TotalHit)
	}
	if s.SavedTokens != 200 {
		t.Errorf("SavedTokens = %d, want 200", s.SavedTokens)
	}
	if s.Hits != 2 {
		t.Errorf("Hits = %d, want 2", s.Hits)
	}
	if s.Misses != 1 {
		t.Errorf("Misses = %d, want 1", s.Misses)
	}
}

func TestNameDelegatesToInner(t *testing.T) {
	mp := &mockProvider{name: "anthropic"}
	ct := NewCacheTracker(mp)

	if got := ct.Name(); got != "anthropic" {
		t.Errorf("Name() = %q, want %q", got, "anthropic")
	}
}

func TestChatErrorPassesThroughWithoutTracking(t *testing.T) {
	errAPI := errors.New("API rate limited")
	mp := &mockProvider{err: errAPI}
	ct := NewCacheTracker(mp)

	_, err := ct.Chat([]Message{{Role: "user", Content: "hello"}}, nil)
	if !errors.Is(err, errAPI) {
		t.Errorf("expected error %v, got %v", errAPI, err)
	}

	s := ct.Stats()
	if s.Requests != 0 {
		t.Errorf("Requests = %d, want 0 (errors should not be tracked)", s.Requests)
	}
}

func TestFingerprintChangesWhenMessagesChange(t *testing.T) {
	msgs1 := []Message{{Role: "user", Content: "hello"}}
	msgs2 := []Message{{Role: "user", Content: "goodbye"}}

	fp1 := fingerprint(msgs1, nil)
	fp2 := fingerprint(msgs2, nil)

	if fp1 == fp2 {
		t.Error("fingerprint should differ for different messages")
	}
}

func TestFingerprintStableForSameInput(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "hello"},
	}
	tools := []ToolDef{{Name: "read_file"}}

	fp1 := fingerprint(msgs, tools)
	fp2 := fingerprint(msgs, tools)

	if fp1 != fp2 {
		t.Errorf("fingerprint not stable: %d != %d", fp1, fp2)
	}
}
