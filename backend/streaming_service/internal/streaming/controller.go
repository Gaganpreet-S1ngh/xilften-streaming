package streaming

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

func (h *Handler) CreateMovieHandler(c *gin.Context) {
	var movie CreateAndUpdateMovieRequest

	if err := c.ShouldBindJSON(&movie); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// Create timeout context as a child of req context

	movieID, err := h.svc.CreateMovie(c.Request.Context(), movie)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	success(c, http.StatusOK, movieID)
}

func (h *Handler) DeleteMovieHandler(c *gin.Context) {
	movieID := c.Param("id")

	movieUUID, err := uuid.Parse(movieID)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.DeleteMovie(c.Request.Context(), movieUUID); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	success(c, http.StatusOK, "Movie deleted successfully")
}

func (h *Handler) GetMovieHandler(c *gin.Context) {
	movieID := c.Param("id")

	movieUUID, err := uuid.Parse(movieID)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	movie, err := h.svc.GetMovieByID(c.Request.Context(), movieUUID)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	success(c, http.StatusOK, movie)
}

func (h *Handler) UpdateMovieHandler(c *gin.Context) {
	movieID := c.Param("id")

	movieUUID, err := uuid.Parse(movieID)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var movie CreateAndUpdateMovieRequest

	if err := c.ShouldBindJSON(&movie); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// Create timeout context as a child of req context

	if err := h.svc.UpdateMovie(c.Request.Context(), movieUUID, movie); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	success(c, http.StatusOK, "Movie updated successfully!")
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
