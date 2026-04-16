package user

import (
	"net/http"
	"time"

	pkg "github.com/Gaganpreet-S1ngh/xilften-user-service/pkg/auth"
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

func NewRoutes(ginEngine *gin.Engine, handler *Handler, auth pkg.Auth) Routes {
	return &routes{
		ginEngine: ginEngine,
		handler:   handler,
		auth:      auth,
	}
}

// SetupPrivateRoutes implements [Routes].
func (r *routes) SetupPrivateRoutes() {
	r.ginEngine.GET("/auth/logout", Authenticate(r.auth), r.handler.LogoutHandler)
}

// SetupPublicRoutes implements [Routes].
func (r *routes) SetupPublicRoutes() {

	r.ginEngine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "OK",
			"time":   time.Now().UTC(),
		})
	})

	r.ginEngine.POST("/auth/register", r.handler.RegisterHandler)
	r.ginEngine.POST("/auth/login", r.handler.LoginHandler)

}
