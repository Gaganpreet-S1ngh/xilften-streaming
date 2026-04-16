package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/zap"
)

type SQLDatabase interface {
	GetDBClient() *bun.DB
	Connect(ctx context.Context, dsn string) error
	RegisterModels(ctx context.Context, models ...any) error
}

type sqlDatabase struct {
	dbClient *bun.DB
	logger   *zap.Logger
}

func NewSQLDatabase(logger *zap.Logger) SQLDatabase {
	return &sqlDatabase{
		logger: logger,
	}
}

// Connect implements [SQLDatabase].
func (s *sqlDatabase) Connect(ctx context.Context, dsn string) error {
	var err error
	maxRetries := 5
	baseDelay := 2 * time.Second

	for i := range maxRetries {
		select {
		case <-ctx.Done():
			log.Println("Cancelling database connection due to server shutdown...")
			return nil
		default:
		}

		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		sqldb.SetMaxOpenConns(25)
		sqldb.SetMaxIdleConns(10)
		sqldb.SetConnMaxLifetime(5 * time.Minute)

		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = db.PingContext(pingCtx)
		cancel()

		if err == nil {
			s.logger.Info("database connected")
			s.dbClient = db
			return nil
		}

		s.logger.Warn("database connection failed, retrying",
			zap.Int("attempt", i+1),
			zap.Error(err),
		)

		time.Sleep(baseDelay * time.Duration(i+1)) // exponential-ish backoff
	}

	return fmt.Errorf("failed to connect to database after retries: %w", err)
}

// GetDBClient implements [SQLDatabase].
func (s *sqlDatabase) GetDBClient() *bun.DB {
	return s.dbClient
}

// RegisterModels implements [SQLDatabase].
func (s *sqlDatabase) RegisterModels(ctx context.Context, models ...any) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Register all models (important for relations like m2m)
	for _, model := range models {
		s.dbClient.RegisterModel(model)
	}

	// Create tables
	for _, model := range models {
		if _, err := s.dbClient.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx); err != nil {

			s.logger.Error("failed to init table",
				zap.String("model", fmt.Sprintf("%T", model)),
				zap.Error(err),
			)
			return err
		}
	}

	s.logger.Info("database schema initialized")
	return nil
}
