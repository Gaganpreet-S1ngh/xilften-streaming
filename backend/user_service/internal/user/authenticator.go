package user

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	pkg "github.com/Gaganpreet-S1ngh/xilften-user-service/pkg/auth"
	"github.com/gin-gonic/gin"
)

// Lets keep it simple for simple auth operations like browsing the website
func Authenticate(a pkg.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := extractBearerToken(c)
		if err != nil {
			abortUnauthorized(c, err.Error())
			return
		}

		claims, err := a.VerifyAccessToken(tokenStr)
		if err != nil {
			abortUnauthorized(c, err.Error())
			return
		}

		// Only accept access token not refresh token
		if claims.TokenType != "access" {
			abortUnauthorized(c, "Token type is not permitted on this endpoint!")
		}

		// Inject inside gin context
		c.Set("claims", claims)
		c.Set("user_id", claims.UserID)
		c.Set("session_id", claims.SessionID)
		c.Set("email", claims.Email)
		c.Set("user_type", claims.UserType)

		c.Next()
	}
}

// For stricter operations such that updating profile or placing order or etc we should verify if the token is revoked or not
// So that even if someone steals the token the damage can be minimized until the access token is valid
func AuthenticateWithSession(a pkg.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := extractBearerToken(c)
		if err != nil {
			abortUnauthorized(c, err.Error())
			return
		}

		claims, err := a.VerifyAccessToken(tokenStr)
		if err != nil {
			abortUnauthorized(c, err.Error())
			return
		}

		// Only accept access token not refresh token
		if claims.TokenType != "access" {
			abortUnauthorized(c, "Token type is not permitted on this endpoint!")
		}

		// Send session ID in header or in JWT?
		sessionID := strings.TrimSpace(c.GetHeader("X-Session-ID"))

		if len(sessionID) == 0 {
			abortUnauthorized(c, "Missing Session ID!")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		refreshToken := strings.TrimSpace(c.GetHeader("X-Refresh-Token"))

		userSession, err := a.GetSession(ctx, sessionID, refreshToken)
		if err != nil {
			abortUnauthorized(c, err.Error())
		}

		// Cross verify token owner and session matches
		if userSession.UserID != claims.UserID {
			abortUnauthorized(c, "Session doesn't match with the authenticated user!")
			return
		}

		c.Set("claims", claims)
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("user_type", claims.UserType)
		c.Set("session_id", sessionID)

		c.Next()

	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make([]string, len(roles))

	for _, r := range roles {
		allowed = append(allowed, strings.ToLower(r))
	}

	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists {
			abortUnauthorized(c, "User type not found - make sure to authenticate first!")
			return
		}

		ut, ok := userType.(string)
		if !ok {
			abortForbidden(c, "Malformed user type claims!")
			return
		}

		for _, allowedUser := range allowed {
			if ut == allowedUser {
				c.Next()
			}
		}

		abortForbidden(c, fmt.Sprintf("Access denied: role (%s) is not authorized for this resource", ut))
	}

}

//=================================//
//		 HELPER FUNCTION           //
//=================================//

// Extract bearer token from the header
func extractBearerToken(c *gin.Context) (string, error) {
	authHeaderValue := c.GetHeader("Authorization")

	if len(authHeaderValue) == 0 {
		return "", fmt.Errorf("Missing authorization header!")
	}

	// Bearer token
	tokenSplit := strings.Split(authHeaderValue, " ")

	if len(tokenSplit) != 2 || tokenSplit[0] != "Bearer" {
		return "", fmt.Errorf("Authorization header must be in format : Bearer <token>")
	}

	token := strings.TrimSpace(tokenSplit[1])

	if len(token) == 0 {
		return "", fmt.Errorf("Token is empty!")
	}

	return token, nil
}

func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error":   message,
		"code":    "UNAUTHORIZED",
	})
}

func abortForbidden(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"error":   message,
		"code":    "FORBIDDEN",
	})
}
