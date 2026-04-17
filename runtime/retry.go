package runtime

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

// RetryPolicy controls provider-call retry behavior.
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Jitter      bool

	// IsRetryable determines whether to retry on a given error.
	IsRetryable func(err error) bool
}

// DefaultRetryPolicy returns a sensible default policy.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Second,
		MaxDelay:    32 * time.Second,
		Jitter:      true,
		IsRetryable: func(err error) bool {
			// Retry on rate limit and transient 5xx
			var rle *RateLimitError
			if errors.As(err, &rle) {
				return true
			}
			var ase *APIStatusError
			if errors.As(err, &ase) && ase.StatusCode >= 500 {
				return true
			}
			return false
		},
	}
}

// Delay returns the retry delay for a given attempt number (0-indexed).
func (p RetryPolicy) Delay(attempt int) time.Duration {
	d := float64(p.BaseDelay) * math.Pow(2, float64(attempt))
	if d > float64(p.MaxDelay) {
		d = float64(p.MaxDelay)
	}
	if p.Jitter {
		d = d * (0.5 + rand.Float64()) // 0.5x–1.5x jitter
	}
	return time.Duration(d)
}

// RateLimitError is returned by providers on 429.
type RateLimitError struct {
	RetryAfter time.Duration
	Msg        string
}

func (e *RateLimitError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return "rate limit exceeded"
}

// APIStatusError wraps a non-2xx response.
type APIStatusError struct {
	StatusCode int
	Msg        string
}

func (e *APIStatusError) Error() string {
	return e.Msg
}
