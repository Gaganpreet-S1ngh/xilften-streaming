package contracts

type Ranking struct {
	IMDbRating     float32 `bun:"imdb_rating" json:"imdb_rating"`
	RottenTomatoes int     `bun:"rotten_tomatoes" json:"rotten_tomatoes"`
	Popularity     float32 `bun:"popularity" json:"popularity"`
}
