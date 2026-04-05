package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisDatabase interface {
	GetClient() *redis.Client
	Connect(ctx context.Context, dsn string) error
	Close() error
	Ping(ctx context.Context) error
}

type redisDatabase struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisDatabase(logger *zap.Logger) RedisDatabase {
	return &redisDatabase{
		logger: logger,
	}
}

// Connect implements [RedisDatabase].
func (r *redisDatabase) Connect(ctx context.Context, dsn string) error {
	opts, err := redis.ParseURL(dsn)
	if err != nil {
		return fmt.Errorf("invalid redis DSN: %w", err)
	}

	opts.PoolSize = 25
	opts.MinIdleConns = 5
	opts.MaxIdleConns = 10
	opts.ConnMaxLifetime = 5 * time.Minute
	opts.ConnMaxIdleTime = 2 * time.Minute

	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second

	maxRetries := 5
	baseDelay := 2 * time.Second

	for i := range maxRetries {
		select {
		case <-ctx.Done():
			r.logger.Info("cancelling redis connection due to server shutdown")
			return nil
		default:
		}

		client := redis.NewClient(opts)

		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = client.Ping(pingCtx).Err()
		cancel()

		if err == nil {
			r.logger.Info("redis connected")
			r.client = client
			return nil
		}

		// Clean up the failed client before retrying.
		_ = client.Close()

		r.logger.Warn("redis connection failed, retrying",
			zap.Int("attempt", i+1),
			zap.Error(err),
		)

		delay := baseDelay * time.Duration(i+1) // exponential-ish backoff
		select {
		case <-ctx.Done():
			r.logger.Info("cancelling redis connection due to server shutdown")
			return nil
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("failed to connect to redis after %d retries: %w", maxRetries, err)
}

// Close implements [RedisDatabase].
func (r *redisDatabase) Close() error {
	if r.client == nil {
		return nil
	}
	if err := r.client.Close(); err != nil {
		r.logger.Error("failed to close redis connection", zap.Error(err))
		return fmt.Errorf("failed to close redis connection: %w", err)
	}
	r.logger.Info("redis connection closed")
	return nil
}

// GetClient implements [RedisDatabase].
func (r *redisDatabase) GetClient() *redis.Client {
	return r.client
}

// Ping implements [RedisDatabase].
func (r *redisDatabase) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := r.client.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}
