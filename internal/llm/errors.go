package llm

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"
)

// ProviderError is a typed error from an LLM provider HTTP call.
// It classifies failures as retryable (transient) or fatal (permanent).
type ProviderError struct {
	Provider   string // "anthropic", "openai", etc.
	StatusCode int    // HTTP status code (0 for network errors)
	Message    string // human-readable error message
	ErrorType  string // provider-specific error type (e.g. "rate_limit_error")
	RetryAfter time.Duration // from Retry-After header, 0 if absent
	Wrapped    error  // underlying error (network, JSON parse, etc.)
}

func (e *ProviderError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s: http %d: %s", e.Provider, e.StatusCode, e.Message)
	}
	if e.Wrapped != nil {
		return fmt.Sprintf("%s: %s: %s", e.Provider, e.Message, e.Wrapped.Error())
	}
	return fmt.Sprintf("%s: %s", e.Provider, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Wrapped
}

// Retryable returns true if this error is transient and the request should be retried.
// Retryable: 429 (rate limit), 500+ (server error), network errors.
// Fatal: 400 (bad request), 401 (auth), 403 (forbidden), 404 (not found).
func (e *ProviderError) Retryable() bool {
	// Network error (no status code) — always retryable.
	if e.StatusCode == 0 && e.Wrapped != nil {
		return true
	}
	switch e.StatusCode {
	case http.StatusTooManyRequests: // 429
		return true
	case http.StatusInternalServerError, // 500
		http.StatusBadGateway,        // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout:     // 504
		return true
	default:
		return false
	}
}

// IsRetryable checks if an error is a retryable ProviderError.
func IsRetryable(err error) bool {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.Retryable()
	}
	return false
}

// GetRetryAfter extracts the RetryAfter duration from a ProviderError, if present.
func GetRetryAfter(err error) time.Duration {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.RetryAfter
	}
	return 0
}

// RetryConfig controls the retry behavior for provider calls.
type RetryConfig struct {
	MaxAttempts int           // total attempts (1 = no retry)
	BaseDelay   time.Duration // initial backoff delay
	MaxDelay    time.Duration // cap on backoff delay
}

// DefaultRetryConfig is the default retry configuration for provider calls.
var DefaultRetryConfig = RetryConfig{
	MaxAttempts: 4, // 1 initial + 3 retries
	BaseDelay:   time.Second,
	MaxDelay:    8 * time.Second,
}

// RetryDo executes fn with exponential backoff on retryable errors.
// It respects Retry-After headers when present.
func RetryDo(cfg RetryConfig, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !IsRetryable(lastErr) {
			return lastErr
		}
		// Don't sleep after the last attempt.
		if attempt == cfg.MaxAttempts-1 {
			break
		}
		// Use Retry-After if the provider sent one, otherwise exponential backoff.
		delay := GetRetryAfter(lastErr)
		if delay == 0 {
			delay = cfg.BaseDelay * time.Duration(math.Pow(2, float64(attempt)))
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
		time.Sleep(delay)
	}
	return &ProviderError{
		Provider: "retry",
		Message:  fmt.Sprintf("exhausted %d attempts", cfg.MaxAttempts),
		Wrapped:  lastErr,
	}
}
