package streaming

import (
	"net/http"
	"time"

	pkg "github.com/Gaganpreet-S1ngh/xilften-streaming-service/internal/pkg/auth"
	"github.com/Gaganpreet-S1ngh/xilften-streaming-service/internal/user"
	"github.com/gin-gonic/gin"
)

type Routes interface {
	SetupPublicRoutes()
	SetupPrivateRoutes()
}

type routes struct {
	ginEngine *gin.Engine
	handler   *Handler
	auth      pkg.Auth
}

func NewRoutes(ginEngine *gin.Engine, auth pkg.Auth, handler *Handler) Routes {
	return &routes{
		ginEngine: ginEngine,
		auth:      auth,
		handler:   handler,
	}
}

// SetupPrivateRoutes implements [Routes].
func (r *routes) SetupPrivateRoutes() {
	panic("unimplemented")
}

// SetupPublicRoutes implements [Routes].
func (r *routes) SetupPublicRoutes() {

	r.ginEngine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "OK",
			"time":   time.Now().UTC(),
		})
	})

	r.ginEngine.POST("/movies", r.handler.CreateMovieHandler)
	r.ginEngine.GET("/movies", user.Authenticate(r.auth), user.RequireRole("admin"), r.handler.GetMoviesHandler)
	r.ginEngine.GET("/movies/:id", r.handler.GetMovieHandler)
	r.ginEngine.PATCH("/movies/:id", r.handler.UpdateMovieHandler)
	r.ginEngine.DELETE("/movies/:id", r.handler.DeleteMovieHandler)

	r.ginEngine.GET("/movies/genre", r.handler.GetGenresHandler)
	r.ginEngine.DELETE("/movies/genre/:id", r.handler.DeleteGenreHandler)
	r.ginEngine.POST("/movies/genre", r.handler.CreateGenreHandler)

}
