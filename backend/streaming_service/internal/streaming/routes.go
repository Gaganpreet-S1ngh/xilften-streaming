package streaming

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Routes interface {
	SetupPublicRoutes()
	SetupPrivateRoutes()
}

type routes struct {
	ginEngine *gin.Engine
	handler   *Handler
}

func NewRoutes(ginEngine *gin.Engine, handler *Handler) Routes {
	return &routes{
		ginEngine: ginEngine,
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
	r.ginEngine.GET("/movies", r.handler.GetMoviesHandler)
	r.ginEngine.GET("/movies/:id", r.handler.GetMovieHandler)
	r.ginEngine.PATCH("/movies/:id", r.handler.UpdateMovieHandler)
	r.ginEngine.DELETE("/movies/:id", r.handler.DeleteMovieHandler)

}
