package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type Repository interface {
	Create(ctx context.Context, user *User) (uuid.UUID, error)
	Update(ctx context.Context, userID uuid.UUID, updates map[string]any) (*User, error)
	Delete(ctx context.Context, userID uuid.UUID) error
	FindByID(ctx context.Context, userID uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	GetUsers(ctx context.Context, limit, offset int) ([]*User, error)
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

// Create implements [Repository].
func (r *repository) Create(ctx context.Context, user *User) (uuid.UUID, error) {
	err := r.executeWithRetry(ctx, func() error {
		_, err := r.db.NewInsert().
			Model(user).
			Returning("*"). // Populates the user
			Exec(ctx)

		return err
	}, "Create User")

	if err != nil {
		return uuid.Nil, err
	}

	return user.ID, nil
}

// Delete implements [Repository].
func (r *repository) Delete(ctx context.Context, userID uuid.UUID) error {
	return r.executeWithRetry(ctx, func() error {
		res, err := r.db.NewDelete().
			Model((*User)(nil)).
			Where("id = ?", userID).
			Exec(ctx)

		if err != nil {
			return fmt.Errorf("Error deleting user (%s) : %w", userID.String(), err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("Error deleting user (%s) : %w", userID.String(), err)
		}

		if affected == 0 {
			return nil
		}

		r.logger.Info("User deleted", zap.String("user_id", userID.String()))
		return nil
	}, "Delete User")
}

// FindByEmail implements [Repository].
func (r *repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	user := new(User)
	err := r.executeWithRetry(ctx, func() error {
		return r.db.NewSelect().
			Model(user).
			Where("email = ?", strings.ToLower(email)).
			Scan(ctx)
	}, "Find User By Email")

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("Error finding user with email (%s) : %w", email, err)
	}

	user.Password = ""
	return user, nil
}

// FindByID implements [Repository].
func (r *repository) FindByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	user := new(User)
	err := r.executeWithRetry(ctx, func() error {
		return r.db.NewSelect().
			Model(user).
			Where("u.id = ?", userID).
			Relation("FavouriteGenres").
			Scan(ctx)
	}, "Find User By ID")

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("Error finding user with ID (%s) : %w", userID.String(), err)
		}
		return nil, err
	}

	user.Password = ""

	return user, nil
}

// GetUsers implements [Repository].
func (r *repository) GetUsers(ctx context.Context, limit int, offset int) ([]*User, error) {
	var users []*User
	err := r.executeWithRetry(ctx, func() error {
		return r.db.NewSelect().
			Model(&users).
			Order("created_at DESC").
			Limit(limit).
			Offset(offset).
			Scan(ctx)
	}, "Get Users")

	if err != nil {
		return nil, fmt.Errorf("Error fetching users : %w", err)
	}

	for _, u := range users {
		u.Password = ""
	}

	return users, nil
}

// Update implements [Repository].
func (r *repository) Update(ctx context.Context, userID uuid.UUID, updates map[string]any) (*User, error) {
	user := new(User)

	err := r.executeWithRetry(ctx, func() error {
		query := r.db.NewUpdate().
			Model(user).
			Where("id = ?", userID)

		updates["updated_at"] = time.Now()

		for key, value := range updates {
			query = query.Set(key+" = ?", value)
		}

		_, err := query.Returning("*").Exec(ctx)
		return err
	}, "Update User")

	if err != nil {
		return nil, fmt.Errorf("Error updating user with ID (%s) : %w", userID.String(), err)
	}

	user.Password = ""
	return user, nil
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
