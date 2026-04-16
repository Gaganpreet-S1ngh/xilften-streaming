package streaming

import (
	"time"

	"github.com/Gaganpreet-S1ngh/xilften-streaming-service/internal/contracts"
	"github.com/google/uuid"
)

type CreateAndUpdateMovieRequest struct {
	ImdbID           string            `json:"imdb_id" validate:"required"`
	Title            string            `json:"title" validate:"required"`
	PosterPath       string            `json:"poster_path"`
	YoutubeTrailerID string            `json:"youtube_trailer_id"`
	AdminReview      string            `json:"admin_review"`
	Ranking          contracts.Ranking `json:"ranking"`
	GenreIDs         []uuid.UUID       `json:"genre_ids"`
}

type GetMovieResponse struct {
	ID               uuid.UUID         `json:"id"`
	ImdbID           string            `json:"imdb_id"`
	Title            string            `json:"title"`
	PosterPath       string            `json:"poster_path"`
	YoutubeTrailerID string            `json:"youtube_trailer_id"`
	AdminReview      string            `json:"admin_review"`
	Ranking          contracts.Ranking `json:"ranking"`
	Genres           []GenreDTO        `json:"genres"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type GenreDTO struct {
	ID        string    `json:"id"`
	GenreName string    `json:"genre_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
