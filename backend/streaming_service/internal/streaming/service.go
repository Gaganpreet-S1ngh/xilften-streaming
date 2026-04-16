package streaming

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service interface {
	CreateMovie(ctx context.Context, movieDetails CreateAndUpdateMovieRequest) (uuid.UUID, error)
	UpdateMovie(ctx context.Context, movieID uuid.UUID, movieDetails CreateAndUpdateMovieRequest) error
	DeleteMovie(ctx context.Context, movieID uuid.UUID) error
	GetMovieByID(ctx context.Context, movieID uuid.UUID) (GetMovieResponse, error)
	GetMovies(ctx context.Context, limit int, offset int) ([]GetMovieResponse, error)
}

type service struct {
	repository Repository
	logger     *zap.Logger
}

func NewService(repository Repository, logger *zap.Logger) Service {
	return &service{
		repository: repository,
		logger:     logger,
	}
}

// CreateMovie implements [Service].
func (s *service) CreateMovie(ctx context.Context, movieDetails CreateAndUpdateMovieRequest) (uuid.UUID, error) {
	// Check if the imdb id of this movie exists
	existingMovie, err := s.repository.FindMovieByImdbID(ctx, movieDetails.ImdbID)
	// if err != nil {
	// 	return uuid.Nil, err
	// }

	if existingMovie != nil {
		return uuid.Nil, fmt.Errorf("movie with IMDB ID (%s) already exists", movieDetails.ImdbID)
	}

	// Adding genre from dropdown hence no need to check? Or prevent from MITM

	genres := make([]*Genre, len(movieDetails.GenreIDs))
	for index, id := range movieDetails.GenreIDs {
		genres[index] = &Genre{ID: id} // bun only needs the PK for m2m inserts
	}

	movie := &Movie{
		ImdbID:           movieDetails.ImdbID,
		Title:            movieDetails.Title,
		PosterPath:       movieDetails.PosterPath,
		YoutubeTrailerID: movieDetails.YoutubeTrailerID,
		AdminReview:      movieDetails.AdminReview,
		Ranking:          movieDetails.Ranking,
		Genres:           genres,
	}

	movieID, err := s.repository.CreateMovie(ctx, movie)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create movie: %w", err)
	}

	if len(movieDetails.GenreIDs) > 0 {
		if err := s.repository.ReplaceGenresForMovie(ctx, movieID, movieDetails.GenreIDs); err != nil {
			return uuid.Nil, fmt.Errorf("movie created but failed to attach genres: %w", err)
		}
	}

	return movieID, nil
}

// DeleteMovie implements [Service].
func (s *service) DeleteMovie(ctx context.Context, movieID uuid.UUID) error {

	if err := s.repository.DeleteMovie(ctx, movieID); err != nil {
		return err
	}
	return nil
}

// GetMovieByID implements [Service].
func (s *service) GetMovieByID(ctx context.Context, movieID uuid.UUID) (GetMovieResponse, error) {
	movie, err := s.repository.FindMovieByID(ctx, movieID)
	if err != nil {
		return GetMovieResponse{}, err
	}

	// Get genres

	genres, err := s.repository.GetGenresForMovie(ctx, movieID)
	if err != nil {
		return GetMovieResponse{}, err
	}

	formattedGenres := make([]GenreDTO, len(genres))

	for _, genre := range genres {
		formattedGenres = append(formattedGenres, GenreDTO{
			ID:        genre.ID.String(),
			GenreName: genre.GenreName,
			CreatedAt: genre.CreatedAt,
			UpdatedAt: genre.UpdatedAt,
		})
	}

	response := GetMovieResponse{
		ID:               movie.ID,
		ImdbID:           movie.ImdbID,
		Title:            movie.Title,
		PosterPath:       movie.PosterPath,
		YoutubeTrailerID: movie.YoutubeTrailerID,
		AdminReview:      movie.AdminReview,
		Ranking:          movie.Ranking,
		Genres:           formattedGenres,
	}

	return response, nil
}

// UpdateMovie implements [Service].
func (s *service) UpdateMovie(ctx context.Context, movieID uuid.UUID, movieDetails CreateAndUpdateMovieRequest) error {
	// Adding genre from dropdown hence no need to check? Or prevent from MITM

	genres := make([]*Genre, len(movieDetails.GenreIDs))
	for index, id := range movieDetails.GenreIDs {
		genres[index] = &Genre{ID: id} // bun only needs the PK for m2m inserts
	}

	movie := &Movie{
		ImdbID:           movieDetails.ImdbID,
		Title:            movieDetails.Title,
		PosterPath:       movieDetails.PosterPath,
		YoutubeTrailerID: movieDetails.YoutubeTrailerID,
		AdminReview:      movieDetails.AdminReview,
		Ranking:          movieDetails.Ranking,
		Genres:           genres,
	}

	err := s.repository.UpdateMovie(ctx, movieID, movie)
	if err != nil {
		return fmt.Errorf("failed to update movie: %w", err)
	}

	if len(movieDetails.GenreIDs) > 0 {
		if err := s.repository.ReplaceGenresForMovie(ctx, movieID, movieDetails.GenreIDs); err != nil {
			return fmt.Errorf("movie created but failed to attach genres: %w", err)
		}
	}

	return nil
}

// GetMovies implements [Service].
func (s *service) GetMovies(ctx context.Context, limit int, offset int) ([]GetMovieResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	movies, err := s.repository.GetMovies(ctx, limit, offset)

	if err != nil {
		s.logger.Error("failed to get movies", zap.Error(err))
		return nil, err
	}

	// Map pointer field to response
	response := make([]GetMovieResponse, 0, len(movies))

	// Keep it lightweight for list apis as we are copying structs from pointer of database and other not needed
	// Keep genres if needed now its not so skip in response and mapping
	for _, movie := range movies {
		response = append(response, GetMovieResponse{
			ID:         movie.ID,
			Title:      movie.Title,
			PosterPath: movie.PosterPath,
			Ranking:    movie.Ranking,
		})
	}

	return response, nil
}
