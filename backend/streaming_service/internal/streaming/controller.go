package streaming

import (
	"net/http"
	"strconv"

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

func (h *Handler) GetMoviesHandler(c *gin.Context) {
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	// Convert to int

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// Create timeout context as a child of req context

	movies, err := h.svc.GetMovies(c.Request.Context(), limit, offset)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	success(c, http.StatusOK, movies)
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
