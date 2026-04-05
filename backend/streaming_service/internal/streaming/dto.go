package streaming

import (
	"time"

	"github.com/google/uuid"
)

type GetMovieResponse struct {
	ID               uuid.UUID  `json:"id"`
	ImdbID           string     `json:"imdb_id"`
	Title            string     `json:"title"`
	PosterPath       string     `json:"poster_path"`
	YoutubeTrailerID string     `json:"youtube_trailer_id"`
	AdminReview      string     `json:"admin_review"`
	Ranking          RankingDTO `json:"ranking"`
	Genres           []GenreDTO `json:"genres"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type GenreDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RankingDTO struct {
	IMDbRating     float32 `json:"imdb_rating"`
	RottenTomatoes int     `json:"rotten_tomatoes"`
	Popularity     float32 `json:"popularity"`
}
