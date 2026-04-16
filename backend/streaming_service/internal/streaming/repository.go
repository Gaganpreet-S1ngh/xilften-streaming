package streaming

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
	CreateMovie(ctx context.Context, movie *Movie) (uuid.UUID, error)
	UpdateMovie(ctx context.Context, movieID uuid.UUID, updatedMovie *Movie) error
	DeleteMovie(ctx context.Context, movieID uuid.UUID) error
	FindMovieByID(ctx context.Context, movieID uuid.UUID) (*Movie, error)
	FindMovieByImdbID(ctx context.Context, imdbID string) (*Movie, error)
	GetMovies(ctx context.Context, limit int, offset int) ([]*Movie, error)

	CreateGenre(ctx context.Context, genre *Genre) (uuid.UUID, error)
	DeleteGenre(ctx context.Context, genreID uuid.UUID) error
	FindGenreByID(ctx context.Context, genreID uuid.UUID) (*Genre, error)
	GetGenres(ctx context.Context, limit int, offset int) ([]*Genre, error)

	AddGenreToMovie(ctx context.Context, movieID uuid.UUID, genreID uuid.UUID) error
	RemoveGenreFromMovie(ctx context.Context, movieID uuid.UUID, genreID uuid.UUID) error
	RemoveAllGenresFromMovie(ctx context.Context, movieID uuid.UUID) error
	GetGenresForMovie(ctx context.Context, movieID uuid.UUID) ([]*Genre, error)
	GetMoviesForGenre(ctx context.Context, genreID uuid.UUID) ([]*Movie, error)
	ReplaceGenresForMovie(ctx context.Context, movieID uuid.UUID, genreIDs []uuid.UUID) error
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
//        MOVIE CRUD OPERATIONS             //
//==========================================//

// Create implements [Repository].
func (r *repository) CreateMovie(ctx context.Context, movie *Movie) (uuid.UUID, error) {
	err := r.executeWithRetry(ctx, func() error {
		_, err := r.db.NewInsert().
			Model(movie).
			Returning("*").
			Exec(ctx)

		return err
	}, "Create Movie")

	if err != nil {
		return uuid.Nil, fmt.Errorf("Error creating movie : %w", err)
	}

	return movie.ID, nil
}

// Delete implements [Repository].
func (r *repository) DeleteMovie(ctx context.Context, movieID uuid.UUID) error {
	return r.executeWithRetry(ctx, func() error {
		return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// 1. Delete the movie

			res, err := r.db.NewDelete().
				Model((*Movie)(nil)).
				Where("_id = ?", movieID).
				Exec(ctx)

			if err != nil {
				return fmt.Errorf("Error deleting movie (%s) : %w", movieID.String(), err)
			}

			affected, err := res.RowsAffected()
			if err != nil {
				return fmt.Errorf("Error deleting movie (%s) : %w", movieID.String(), err)
			}

			if affected == 0 {
				return fmt.Errorf("Movie (%s) not found!", movieID.String())
			}

			r.logger.Info("Movie deleted", zap.String("movie_id", movieID.String()))

			// 2. Remove all the links of the movie with genre

			if _, err := tx.NewDelete().
				Model((*MovieGenre)(nil)).
				Where("movie_id = ?", movieID).
				Exec(ctx); err != nil {
				return fmt.Errorf("Movies deleted but failed to remove genre links from movie : %w", err)
			}

			return nil

		})

	}, "Delete Movie")
}

// FindByID implements [Repository].
func (r *repository) FindMovieByID(ctx context.Context, movieID uuid.UUID) (*Movie, error) {
	movie := new(Movie)

	err := r.executeWithRetry(ctx, func() error {
		return r.db.NewSelect().
			Model(movie).
			Where("m._id = ?", movieID).
			Relation("Genres").
			Scan(ctx)
	}, "Find Movie By ID")

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("Movie (%s) not found!", movieID.String())
		}
		return nil, fmt.Errorf("Error finding movie with ID (%s) : %w", movieID.String(), err)
	}

	return movie, nil
}

// Update implements [Repository].
func (r *repository) UpdateMovie(ctx context.Context, movieID uuid.UUID, updatedMovie *Movie) error {
	err := r.executeWithRetry(ctx, func() error {
		res, err := r.db.NewUpdate().
			Model(updatedMovie).
			Column(
				"imdb_id",
				"title",
				"poster_path",
				"youtube_trailer_id",
				"admin_review",
				"imdb_rating",
				"rotten_tomatoes",
				"popularity",
			).
			Where("_id = ?", movieID).
			Exec(ctx)

		if err != nil {
			return fmt.Errorf("Error updating movie (%s) : %w", updatedMovie.ID.String(), err)
		}

		rows, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("Error updating movie (%s) : %w", updatedMovie.ID.String(), err)
		}
		if rows == 0 {
			return fmt.Errorf("Movie not found (%d)!")
		}

		return nil
	}, "Update Movie")

	if err != nil {
		return fmt.Errorf("Error updating movie (%s) : %w", updatedMovie.ID.String(), err)
	}

	return nil
}

// GetMovies implements [Repository].
func (r *repository) GetMovies(ctx context.Context, limit int, offset int) ([]*Movie, error) {
	var movies []*Movie

	err := r.executeWithRetry(ctx, func() error {
		return r.db.NewSelect().Model(&movies).Order("created_at DESC").Relation("Genres").
			Limit(limit).
			Offset(offset).
			Scan(ctx)
	}, "Get Movies")

	if err != nil {
		return nil, fmt.Errorf("Error getting movies : %w", err)
	}

	return movies, nil
}

// FindMovieByImdbID implements [Repository].
func (r *repository) FindMovieByImdbID(ctx context.Context, imdbID string) (*Movie, error) {
	movie := new(Movie)

	err := r.executeWithRetry(ctx, func() error {
		return r.db.NewSelect().
			Model(movie).
			Where("m.imdb_id = ?", imdbID).
			Scan(ctx)
	}, "Find Movie By IMDB ID")

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("Movie with imdb id (%s) not found!", imdbID)
		}
		return nil, fmt.Errorf("Error finding movie with IMDB ID (%s) : %w", imdbID, err)
	}

	return movie, nil
}

//==========================================//
//        GENRE CRUD OPERATIONS             //
//==========================================//

// CreateGenre implements [Repository].
func (r *repository) CreateGenre(ctx context.Context, genre *Genre) (uuid.UUID, error) {
	err := r.executeWithRetry(ctx, func() error {
		_, err := r.db.NewInsert().
			Model(genre).
			Returning("*").
			Exec(ctx)

		return err
	}, "Create Genre")

	if err != nil {
		return uuid.Nil, fmt.Errorf("Error creating genre : %w", err)
	}

	return genre.ID, nil
}

// DeleteGenre implements [Repository].
func (r *repository) DeleteGenre(ctx context.Context, genreID uuid.UUID) error {
	return r.executeWithRetry(ctx, func() error {
		res, err := r.db.NewDelete().
			Model((*Movie)(nil)).
			Where("_id = ?", genreID).
			Exec(ctx)

		if err != nil {
			return fmt.Errorf("Error deleting genre (%s) : %w", genreID.String(), err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("Error deleting genre (%s) : %w", genreID.String(), err)
		}

		if affected == 0 {
			return nil
		}

		r.logger.Info("Genre deleted", zap.String("genre_id", genreID.String()))
		return nil
	}, "Delete Genre")
}

// FindGenreByID implements [Repository].
func (r *repository) FindGenreByID(ctx context.Context, genreID uuid.UUID) (*Genre, error) {
	genre := new(Genre)

	err := r.executeWithRetry(ctx, func() error {
		return r.db.NewSelect().
			Model(genre).
			Where("g._id = ?", genreID).
			Scan(ctx)
	}, "Find Genre By ID")

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("Genre (%s) not found!", genreID.String())
		}
		return nil, fmt.Errorf("Error finding genre with ID (%s) : %w", genreID.String(), err)
	}

	return genre, nil
}

// GetGenres implements [Repository].
func (r *repository) GetGenres(ctx context.Context, limit int, offset int) ([]*Genre, error) {
	var genres []*Genre

	err := r.executeWithRetry(ctx, func() error {
		return r.db.NewSelect().Model(&genres).Order("created_at DESC").
			Limit(limit).
			Offset(offset).
			Scan(ctx)
	}, "Get Genres")

	if err != nil {
		return nil, fmt.Errorf("Error getting genres : %w", err)
	}

	return genres, nil
}

//==========================================//
//        GENRE-MOVIE CRUD OPERATIONS       //
//==========================================//

// AddGenreToMovie implements [Repository].
func (r *repository) AddGenreToMovie(ctx context.Context, movieID uuid.UUID, genreID uuid.UUID) error {
	panic("unimplemented")
}

// RemoveAllGenresFromMovie implements [Repository].
func (r *repository) RemoveAllGenresFromMovie(ctx context.Context, movieID uuid.UUID) error {
	panic("unimplemented")
}

// GetMoviesForGenre implements [Repository].
func (r *repository) GetMoviesForGenre(ctx context.Context, genreID uuid.UUID) ([]*Movie, error) {
	panic("unimplemented")
}

// GetGenreForMovie implements [Repository].
func (r *repository) GetGenresForMovie(ctx context.Context, movieID uuid.UUID) ([]*Genre, error) {
	var genres []*Genre

	err := r.executeWithRetry(ctx, func() error {
		return r.db.NewSelect().Model(&genres).
			Join("JOIN movie_genres AS mg ON mg.genre_id = g._id").
			Where("mg.movie_id = ?", movieID).OrderExpr("g.genre_name ASC").
			Scan(ctx)
	}, "Get Genres for Movie")

	if err != nil {
		return nil, fmt.Errorf("Error getting genres for movie (%s) : %w", movieID.String(), err)
	}

	return genres, nil
}

// RemoveGenreFromMovie implements [Repository].
func (r *repository) RemoveGenreFromMovie(ctx context.Context, movieID uuid.UUID, genreID uuid.UUID) error {
	panic("unimplemented")
}

// ReplaceGenresForMovie implements [Repository].
func (r *repository) ReplaceGenresForMovie(ctx context.Context, movieID uuid.UUID, genreIDs []uuid.UUID) error {
	return r.executeWithRetry(ctx, func() error {
		return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// 1. Remove all current genre links for this movie
			if _, err := tx.NewDelete().
				Model((*MovieGenre)(nil)).
				Where("movie_id = ?", movieID).
				Exec(ctx); err != nil {
				return err
			}

			if len(genreIDs) == 0 {
				return nil
			}

			// 2. Build join rows
			rows := make([]MovieGenre, len(genreIDs))
			for i, gid := range genreIDs {
				rows[i] = MovieGenre{MovieID: movieID, GenreID: gid}
			}

			// 3. Bulk-insert
			_, err := tx.NewInsert().
				Model(&rows).
				Exec(ctx)
			return err
		})
	}, "Replace Genres For Movie")
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
