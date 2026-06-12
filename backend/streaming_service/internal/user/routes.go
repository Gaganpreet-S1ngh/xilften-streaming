package user

import (
	pkg "github.com/Gaganpreet-S1ngh/xilften-streaming-service/internal/pkg/auth"
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
	r.ginEngine.GET("/auth/logout", Authenticate(r.auth), RequireRole("admin"), r.handler.LogoutHandler)
}

// SetupPublicRoutes implements [Routes].
func (r *routes) SetupPublicRoutes() {

	r.ginEngine.POST("/auth/register", r.handler.RegisterHandler)
	r.ginEngine.POST("/auth/login", r.handler.LoginHandler)

}
