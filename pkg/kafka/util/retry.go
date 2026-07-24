package util

import (
	"context"
	"time"
)

// RetryConfig задаёт параметры экспоненциального backoff для повторных попыток.
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

var DefaultCloseRetry = RetryConfig{
	MaxAttempts: 3,
	BaseDelay:   200 * time.Millisecond,
	MaxDelay:    2 * time.Second,
}

// WithRetry вызывает fn до успеха или до исчерпания MaxAttempts.
// Между попытками ждёт BaseDelay*2^attempt (с потолком MaxDelay), либо выходит раньше по ctx.
// Возвращает последнюю полученную ошибку, если все попытки исчерпаны.
func WithRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err

			if attempt == cfg.MaxAttempts-1 {
				break
			}

			delay := cfg.BaseDelay * time.Duration(1<<attempt)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}

			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}

		return nil
	}

	return lastErr
}
