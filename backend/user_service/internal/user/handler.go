package user

import (
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

//==========================================//
//             HELPER FUNCTIONS             //
//==========================================//

func success(c *gin.Context, code int, data any) {
	c.JSON(code, gin.H{"data": data, "request_id": c.GetString("request_id")})
}

func fail(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, gin.H{"error": msg, "request_id": c.GetString("request_id")})
}
