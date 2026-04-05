package llm

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestProviderErrorRetryable429(t *testing.T) {
	e := &ProviderError{Provider: "openai", StatusCode: 429, Message: "rate limited"}
	if !e.Retryable() {
		t.Error("429 should be retryable")
	}
}

func TestProviderErrorRetryable500(t *testing.T) {
	e := &ProviderError{Provider: "openai", StatusCode: 500, Message: "internal error"}
	if !e.Retryable() {
		t.Error("500 should be retryable")
	}
}

func TestProviderErrorRetryable502(t *testing.T) {
	e := &ProviderError{Provider: "openai", StatusCode: 502, Message: "bad gateway"}
	if !e.Retryable() {
		t.Error("502 should be retryable")
	}
}

func TestProviderErrorRetryable503(t *testing.T) {
	e := &ProviderError{Provider: "openai", StatusCode: 503, Message: "service unavailable"}
	if !e.Retryable() {
		t.Error("503 should be retryable")
	}
}

func TestProviderErrorRetryableNetwork(t *testing.T) {
	e := &ProviderError{Provider: "openai", StatusCode: 0, Message: "network error", Wrapped: fmt.Errorf("connection refused")}
	if !e.Retryable() {
		t.Error("network error (status 0 with wrapped) should be retryable")
	}
}

func TestProviderErrorFatal401(t *testing.T) {
	e := &ProviderError{Provider: "openai", StatusCode: 401, Message: "unauthorized"}
	if e.Retryable() {
		t.Error("401 should NOT be retryable")
	}
}

func TestProviderErrorFatal400(t *testing.T) {
	e := &ProviderError{Provider: "openai", StatusCode: 400, Message: "bad request"}
	if e.Retryable() {
		t.Error("400 should NOT be retryable")
	}
}

func TestProviderErrorFatal403(t *testing.T) {
	e := &ProviderError{Provider: "openai", StatusCode: 403, Message: "forbidden"}
	if e.Retryable() {
		t.Error("403 should NOT be retryable")
	}
}

func TestProviderErrorMessage(t *testing.T) {
	e := &ProviderError{Provider: "openai", StatusCode: 429, Message: "rate limited"}
	want := "openai: http 429: rate limited"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestProviderErrorMessageNetwork(t *testing.T) {
	e := &ProviderError{Provider: "anthropic", Message: "network error", Wrapped: fmt.Errorf("timeout")}
	want := "anthropic: network error: timeout"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestProviderErrorUnwrap(t *testing.T) {
	inner := fmt.Errorf("connection reset")
	e := &ProviderError{Provider: "openai", Message: "network", Wrapped: inner}
	if !errors.Is(e, inner) {
		t.Error("Unwrap should expose inner error")
	}
}

func TestIsRetryableWithProviderError(t *testing.T) {
	e := &ProviderError{Provider: "openai", StatusCode: 429, Message: "rate limited"}
	if !IsRetryable(e) {
		t.Error("IsRetryable should return true for 429")
	}
}

func TestIsRetryableWithPlainError(t *testing.T) {
	e := fmt.Errorf("something went wrong")
	if IsRetryable(e) {
		t.Error("IsRetryable should return false for plain error")
	}
}

func TestIsRetryableWithWrappedProviderError(t *testing.T) {
	inner := &ProviderError{Provider: "openai", StatusCode: 500, Message: "server error"}
	e := fmt.Errorf("wrapped: %w", inner)
	if !IsRetryable(e) {
		t.Error("IsRetryable should unwrap and find retryable ProviderError")
	}
}

func TestGetRetryAfter(t *testing.T) {
	e := &ProviderError{Provider: "openai", StatusCode: 429, Message: "rate limited", RetryAfter: 5 * time.Second}
	if got := GetRetryAfter(e); got != 5*time.Second {
		t.Errorf("GetRetryAfter = %v, want 5s", got)
	}
}

func TestGetRetryAfterZero(t *testing.T) {
	e := fmt.Errorf("not a provider error")
	if got := GetRetryAfter(e); got != 0 {
		t.Errorf("GetRetryAfter = %v, want 0", got)
	}
}

func TestRetryDoSucceedsFirst(t *testing.T) {
	calls := 0
	err := RetryDo(RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestRetryDoSucceedsAfterRetry(t *testing.T) {
	calls := 0
	err := RetryDo(RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}, func() error {
		calls++
		if calls < 3 {
			return &ProviderError{Provider: "test", StatusCode: 429, Message: "rate limited"}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil after retries, got %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestRetryDoFatalNoRetry(t *testing.T) {
	calls := 0
	err := RetryDo(RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}, func() error {
		calls++
		return &ProviderError{Provider: "test", StatusCode: 401, Message: "unauthorized"}
	})
	if err == nil {
		t.Fatal("expected error for fatal 401")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (no retry on fatal)", calls)
	}
}

func TestRetryDoExhaustsAttempts(t *testing.T) {
	calls := 0
	err := RetryDo(RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}, func() error {
		calls++
		return &ProviderError{Provider: "test", StatusCode: http.StatusServiceUnavailable, Message: "down"}
	})
	if err == nil {
		t.Fatal("expected error after exhausting attempts")
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
	// Should wrap as "exhausted N attempts".
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestRetryDoRespectsRetryAfter(t *testing.T) {
	start := time.Now()
	calls := 0
	err := RetryDo(RetryConfig{MaxAttempts: 2, BaseDelay: time.Hour, MaxDelay: time.Hour}, func() error {
		calls++
		if calls == 1 {
			return &ProviderError{Provider: "test", StatusCode: 429, Message: "rate limited", RetryAfter: 10 * time.Millisecond}
		}
		return nil
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	// Should have used the 10ms RetryAfter, not the 1h BaseDelay.
	if elapsed > time.Second {
		t.Errorf("elapsed %v — should have used RetryAfter (10ms), not BaseDelay (1h)", elapsed)
	}
}
