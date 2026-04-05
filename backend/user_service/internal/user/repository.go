package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type Repository interface {
}

type repository struct {
	db         *bun.DB
	logger     *zap.Logger
	maxRetries int
	retryDelay time.Duration
}

func NewRepository(db *bun.DB, logger *zap.Logger) Repository {
	return &repository{
		db:         db,
		logger:     logger,
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}
}

//==========================================//
//             HELPER FUNCTIONS             //
//==========================================//

func (r *repository) executeWithRetry(ctx context.Context, operation func() error, name string) error {

	var lastErr error
	for attempt := 1; attempt <= r.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		r.logger.Warn(name+" failed",
			zap.Int("attempt", attempt),
			zap.Error(err),
		)

		if attempt == r.maxRetries {
			break
		}

		if r.shouldRetry(err) {
			// Either cancel or wait
			select {
			case <-time.After(r.retryDelay * time.Duration(attempt)):
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			break
		}
	}

	r.logger.Error(name+" failed after retries", zap.Error(lastErr))
	return lastErr
}

func (r *repository) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	retryable := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"deadlock",
	}

	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	for _, e := range retryable {
		if strings.Contains(err.Error(), e) {
			return true
		}
	}

	return false
}
