package streaming

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Movie struct {
	bun.BaseModel `bun:"table:movies,alias:m"`

	ID               uuid.UUID `bun:"_id,pk,type:uuid,default:gen_random_uuid()"`
	ImdbID           string    `bun:"imdb_id"`
	Title            string    `bun:"title"`
	PosterPath       string    `bun:"poster_path"`
	YoutubeTrailerID string    `bun:"youtube_trailer_id"`
	AdminReview      string    `bun:"admin_review"`
	Ranking          Ranking   `bun:",embed"`

	Genres []*Genre `bun:"m2m:movie_genres,join:Movie=Genre"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

//==========================================//
//               RELATIONS                  //
//==========================================//

type Genre struct {
	bun.BaseModel `bun:"table:genres,alias:g"`
	ID            uuid.UUID `bun:"_id,pk,type:uuid,default:gen_random_uuid()"`
	GenreName     string    `bun:"genre_name,unique"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

//==========================================//
//             CUSTOM TYPE                  //
//==========================================//

type Ranking struct {
	IMDbRating     float32 `bun:"imdb_rating"`
	RottenTomatoes int     `bun:"rotten_tomatoes"` // percentage 0-100
	Popularity     float32 `bun:"popularity"`
}

//==========================================//
//               JOIN TABLE                 //
//==========================================//

type MovieGenre struct {
	bun.BaseModel `bun:"table:movie_genres"`

	MovieID uuid.UUID `bun:"movie_id,pk,notnull"`
	GenreID uuid.UUID `bun:"genre_id,pk,notnull"`

	Movie *Movie `bun:"rel:belongs-to,join:movie_id=_id"`
	Genre *Genre `bun:"rel:belongs-to,join:genre_id=_id"`
}
