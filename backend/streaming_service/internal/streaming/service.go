package streaming

import (
	"context"

	"go.uber.org/zap"
)

type Service interface {
	GetMovies(ctx context.Context, limit int, offset int) ([]GetMovieResponse, error)
}

type service struct {
	repository Repository
	logger     *zap.Logger
}

func NewService(repository Repository, logger *zap.Logger) Service {
	return &service{
		repository: repository,
	}
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
			Ranking:    RankingDTO(movie.Ranking),
		})
	}

	return response, nil
}
