package retry

import (
	"context"
	"fmt"
	"time"
)

type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

type Option func(*Config)

func WithMaxAttempts(attempts int) Option {
	return func(c *Config) {
		c.MaxAttempts = attempts
	}
}

func Do(ctx context.Context, fn func() error, opts ...Option) error {
	cfg := &Config{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt == cfg.MaxAttempts-1 {
			return fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxAttempts, lastErr)
		}

		delay := calculateBackoff(attempt, cfg.BaseDelay, cfg.MaxDelay)

		select {
		case <-time.After(delay):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return lastErr
}

func calculateBackoff(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	// Formula: baseDelay * 2^attempt, capped at maxDelay
	// Examples: 1s, 2s, 4s, 8s, 16s, ...

	delay := baseDelay * time.Duration(1<<uint(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}
