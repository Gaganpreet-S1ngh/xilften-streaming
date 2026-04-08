package pkg

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const (
	AccessExpiry  = 15 * time.Minute
	RefreshExpiry = 7 * 24 * time.Hour
)

type Claims struct {
	jwt.RegisteredClaims
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Email     string `json:"email"`
	UserType  string `json:"user_type"`
	TokenType string `json:"token_type"`
}

type SessionInfo struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	TokenHash string `json:"token_hash,omitempty"`

	DeviceID   string `json:"device_id,omitempty"`
	DeviceName string `json:"device_name,omitempty"`
	OS         string `json:"os,omitempty"`
	Browser    string `json:"browser,omitempty"`
	IP         string `json:"ip,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`

	Country string `json:"country,omitempty"`
	City    string `json:"city,omitempty"`

	// Session lifecycle
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastActiveAt time.Time `json:"last_active_at"`

	// Status / control
	IsActive  bool `json:"is_active"`
	IsRevoked bool `json:"is_revoked"`

	// Security tracking
	LoginMethod string   `json:"login_method,omitempty"` // password, google, otp
	Scope       []string `json:"scope,omitempty"`        // permissions
}

type Auth interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password string, hashed string) error

	GenerateAccessToken(userID string, sessionID string, email string, userType string) (string, error)
	GenerateRefreshToken(userID string, email string, userType string) (string, error)

	VerifyAccessToken(tokenStr string) (*Claims, error)
	VerifyRefreshToken(tokenStr string) (*Claims, error)

	StoreSession(ctx context.Context, refreshToken string, userID string, sessionID string) error
	GetSession(ctx context.Context, sessionID string, refreshToken string) (*SessionInfo, error)
	RotateSession(ctx context.Context, sessionID string, newRefreshToken string) error
	RevokeSession(ctx context.Context, sessionID string, userID string) error
	RevokeAllSessions(ctx context.Context, userID string) error
	ListActiveSessions(ctx context.Context, userID string) ([]SessionInfo, error)

	GenRandomID() string
}

type auth struct {
	redisClient   *redis.Client
	accessSecret  string
	refreshSecret string
}

func NewAuth(redisClient *redis.Client, accessSecret string, refreshSecret string) Auth {
	return &auth{
		redisClient:   redisClient,
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
	}
}

//=================================//
// Password Hashing & Verification //
//=================================//

func (a *auth) HashPassword(password string) (string, error) {
	if len(password) < 6 {
		return "", errors.New("Password to short!")
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("Error hashing the password : %w", err)
	}

	return string(hashedPass), nil
}

func (a *auth) VerifyPassword(password string, hashed string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
	if err != nil {
		return fmt.Errorf("Error verifying the password : %w", err)
	}

	return nil
}

//=================================//
//      JWT TOKEN GENERATION       //
//=================================//

func (a *auth) GenerateAccessToken(userID string, sessionID string, email string, userType string) (string, error) {
	if len(userID) == 0 {
		return "", fmt.Errorf("Missing (User ID) to create token!")
	}
	if len(email) == 0 {
		return "", fmt.Errorf("Missing (Email) to create token!")
	}
	if len(userType) == 0 {
		return "", fmt.Errorf("Missing (User Type) to create token!")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:    userID,
		SessionID: sessionID,
		Email:     email,
		UserType:  userType,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        a.GenRandomID(),
			Issuer:    "Xilftren-Streaming",
			Audience:  []string{"users"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessExpiry)),
		},
	})

	return token.SignedString(a.accessSecret)
}

func (a *auth) GenerateRefreshToken(userID string, email string, userType string) (string, error) {
	if len(userID) == 0 {
		return "", fmt.Errorf("Missing (User ID) to create token!")
	}
	if len(email) == 0 {
		return "", fmt.Errorf("Missing (Email) to create token!")
	}
	if len(userType) == 0 {
		return "", fmt.Errorf("Missing (User Type) to create token!")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:    userID,
		Email:     email,
		UserType:  userType,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        a.GenRandomID(),
			Issuer:    "Xilftren-Streaming",
			Audience:  []string{"users"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(RefreshExpiry)),
		},
	})

	return token.SignedString(a.refreshSecret)
}

//=================================//
//      JWT TOKEN VERIFICATION     //
//=================================//

func (a *auth) VerifyAccessToken(tokenStr string) (*Claims, error) {
	if len(tokenStr) == 0 {
		return nil, fmt.Errorf("Invalid token or not token provided!")
	}

	parsedToken, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.accessSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("Error parsing access token : %w", err)
	}

	claims, ok := parsedToken.Claims.(*Claims)

	if !ok || !parsedToken.Valid {
		return nil, errors.New("Invalid access token claims!")
	}

	return claims, nil
}

func (a *auth) VerifyRefreshToken(tokenStr string) (*Claims, error) {
	if len(tokenStr) == 0 {
		return nil, fmt.Errorf("Invalid token or not token provided!")
	}

	parsedToken, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.refreshSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("Error parsing refresh token : %w", err)
	}

	claims, ok := parsedToken.Claims.(*Claims)

	if !ok || !parsedToken.Valid {
		return nil, errors.New("Invalid refresh token claims!")
	}

	return claims, nil
}

//=================================//
//      REDIS TOKEN MANAGEMENT     //
//=================================//

func (a *auth) StoreSession(ctx context.Context, refreshToken string, userID string, sessionID string) error {
	// Create a session key
	sessionKey := fmt.Sprintf("session:%s", sessionID)
	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)

	// Hash the refresh token (use fast hash)
	hashedToken := a.hashToken(refreshToken)

	// Create a user session

	userSession := SessionInfo{
		SessionID:    sessionID,
		UserID:       userID,
		TokenHash:    hashedToken,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastActiveAt: time.Now(),
		ExpiresAt:    time.Now().Add(RefreshExpiry),
	}

	// Convert to JSON
	data, err := json.Marshal(userSession)
	if err != nil {
		return fmt.Errorf("Error marshalling to JSON : %w", err)
	}

	// Pipeline proper storage to redis to ensure full transaction
	// Pipeline executes fn inside a pipeline.

	cmds, err := a.redisClient.Pipelined(ctx, func(p redis.Pipeliner) error {
		// 1. Store session
		p.Set(ctx, sessionKey, data, RefreshExpiry)
		// 2. Add session id to user_sessions
		p.SAdd(ctx, userSessionsKey, userSession.SessionID)
		// 3. Renew the expiry of the user_sessions := no new login? Delete after 7d
		p.Expire(ctx, userSessionsKey, RefreshExpiry)

		return nil
	})
	// Remove it
	log.Println(cmds)

	if err != nil {
		return fmt.Errorf("Error storing user session : %w", err)
	}

	return nil
}

// Return pointer as we will modify after getting
func (a *auth) GetSession(ctx context.Context, sessionID string, refreshToken string) (*SessionInfo, error) {
	sessionKey := fmt.Sprintf("session:%s", sessionID)
	data, err := a.redisClient.Get(ctx, sessionKey).Bytes()

	if err == redis.Nil {
		return nil, fmt.Errorf("Session not found or expired!")
	}

	if err != nil {
		return nil, fmt.Errorf("Error fetching session: %w", err)
	}

	// As our data is json encoded we marshal it to our struct
	var sessionInfo SessionInfo

	if err := json.Unmarshal(data, &sessionInfo); err != nil {
		return nil, fmt.Errorf("Error unmarshalling data : %w", err)
	}

	// Now session info has the data

	// Verify the refresh token

	if sessionInfo.TokenHash != a.hashToken(refreshToken) {
		return nil, fmt.Errorf("Refresh token mismatch!")
	}

	// Verify if the session is revoked

	if sessionInfo.IsRevoked || !sessionInfo.IsActive {
		return nil, fmt.Errorf("Session revoked or inactive!")
	}

	// Check expiry

	if time.Now().After(sessionInfo.ExpiresAt) {
		return nil, fmt.Errorf("Session expired!")
	}

	return &sessionInfo, nil
}

// Rotate token after every refresh to avoid after refresh token stolen validation
// Every session has different refresh tokens

func (a *auth) RotateSession(ctx context.Context, sessionID string, newRefreshToken string) error {
	sessionKey := fmt.Sprintf("session:%s", sessionID)
	data, err := a.redisClient.Get(ctx, sessionKey).Bytes()

	if err == redis.Nil {
		return fmt.Errorf("Session not found or expired!")
	}

	if err != nil {
		return fmt.Errorf("Error fetching session: %w", err)
	}

	// As our data is json encoded we marshal it to our struct
	var sessionInfo SessionInfo

	if err := json.Unmarshal(data, &sessionInfo); err != nil {
		return fmt.Errorf("Error unmarshalling data : %w", err)
	}

	// Update the data

	sessionInfo.TokenHash = a.hashToken(newRefreshToken)
	sessionInfo.UpdatedAt = time.Now()
	sessionInfo.LastActiveAt = time.Now()
	sessionInfo.ExpiresAt = time.Now().Add(RefreshExpiry)

	// Convert to JSON
	updatedData, err := json.Marshal(sessionInfo)

	if err != nil {
		return fmt.Errorf("Error marshalling to JSON : %w", err)
	}

	return a.redisClient.Set(ctx, sessionKey, updatedData, RefreshExpiry).Err()

}

// Revoke for single device logout

func (a *auth) RevokeSession(ctx context.Context, sessionID string, userID string) error {
	sessionKey := fmt.Sprintf("session:%s", sessionID)
	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)

	_, err := a.redisClient.Pipelined(ctx, func(p redis.Pipeliner) error {
		p.Del(ctx, sessionKey)
		p.SRem(ctx, userSessionsKey, sessionID)
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error revoking session : %w", err)
	}

	return nil
}

// Revoke for all devices logout
func (a *auth) RevokeAllSessions(ctx context.Context, userID string) error {
	userSessionKey := fmt.Sprintf("user_sessions:%s", userID)
	sessionIDs, err := a.redisClient.SMembers(ctx, userSessionKey).Result()

	if err != nil {
		return fmt.Errorf("Error fetching user sessions : %w", err)
	}

	_, err = a.redisClient.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, sid := range sessionIDs {
			sessionKey := fmt.Sprintf("session:%s", sid)
			p.Del(ctx, sessionKey)
		}

		p.Del(ctx, userSessionKey)

		return nil
	})

	if err != nil {
		return fmt.Errorf("Error revoking user sessions : %w", err)
	}

	return nil

}

// Get all active session
func (a *auth) ListActiveSessions(ctx context.Context, userID string) ([]SessionInfo, error) {
	userSessionKey := fmt.Sprintf("user_sessions:%s", userID)
	sessionIDs, err := a.redisClient.SMembers(ctx, userSessionKey).Result()

	if err != nil {
		return nil, fmt.Errorf("Error fetching user sessions : %w", err)
	}

	var userSessions []SessionInfo

	for _, sid := range sessionIDs {
		sessionKey := fmt.Sprintf("session:%s", sid)
		data, err := a.redisClient.Get(ctx, sessionKey).Bytes()

		if err == redis.Nil {
			// Expired naturally, clean up the set
			a.redisClient.SRem(ctx, userSessionKey, sid)
			continue
		}

		if err != nil {
			fmt.Printf("Error getting session (%s) : %v", sid, err)
			continue
		}

		var sessionInfo SessionInfo

		if err := json.Unmarshal(data, &sessionInfo); err != nil {
			fmt.Printf("Error unmarshalling data of session %s : %v", sid, err)
			continue
		}

		// Hide token hash
		sessionInfo.TokenHash = ""
		userSessions = append(userSessions, sessionInfo)
	}

	return userSessions, nil
}

//=================================//
//     AUTHENTICATION MIDDLEWARE   //
//=================================//

//=================================//
//         UTILITY FUNCTIONS       //
//=================================//

func (a *auth) GenRandomID() string {
	randStr := make([]byte, 16)
	io.ReadFull(rand.Reader, randStr)

	return hex.EncodeToString(randStr)
}

func (a *auth) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))

	return hex.EncodeToString(hash[:])
}
