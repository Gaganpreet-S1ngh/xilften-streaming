package user

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID        uuid.UUID `bun:"_id,pk,type:uuid,default:gen_random_uuid()"`
	FirstName string    `bun:"first_name"`
	LastName  string    `bun:"last_name"`
	Email     string    `bun:"email,unique,notnull"`
	Password  string    `bun:"password,notnull"`
	Phone     string    `bun:"phone"`

	UserType   string `bun:"user_type,default:'customer'"`
	Code       string `bun:"code,default:'0'"`
	IsVerified bool   `bun:"is_verified,default:false"`

	DOB             time.Time `bun:"dob,nullzero"`
	ImageURL        string    `bun:"image_url"`
	FavouriteGenres []*Genre  `bun:"m2m:user_genres,join:User=Genre"`

	CreatedAt time.Time `bun:"created_at,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,default:current_timestamp"`
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
//               JOIN TABLE                 //
//==========================================//

type UserGenre struct {
	bun.BaseModel `bun:"table:user_genres"`

	UserID  uuid.UUID `bun:"user_id,pk,notnull"`
	GenreID uuid.UUID `bun:"genre_id,pk,notnull"`

	User  *User  `bun:"rel:belongs-to,join:user_id=_id"`
	Genre *Genre `bun:"rel:belongs-to,join:genre_id=_id"`
}
