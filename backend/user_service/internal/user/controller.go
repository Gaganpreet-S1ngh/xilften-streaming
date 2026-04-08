package user

import (
	"net/http"

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

func (h *Handler) Register(c *gin.Context) {
	var user UserRegisterRequest

	if err := c.ShouldBindJSON(&user); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := h.svc.Register(c.Request.Context(), user)

	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	success(c, http.StatusOK, userID)
}

func (h *Handler) Login(c *gin.Context) {
	var user UserLoginRequest

	if err := c.ShouldBindJSON(&user); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	userDetails, err := h.svc.Login(c.Request.Context(), user, "")

	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	success(c, http.StatusOK, userDetails)
}

func (h *Handler) Logout(c *gin.Context) {
	if err := h.svc.Logout(c.Request.Context(), c.GetString("session_id"), c.GetString("user_id")); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	success(c, http.StatusOK, "User logged out successfully!")
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
