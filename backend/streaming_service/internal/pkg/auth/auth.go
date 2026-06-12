package pkg

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

type Auth interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password string, hashed string) error

	GenerateAccessToken(userID string, sessionID string, email string, userType string) (string, error)
	GenerateRefreshToken(userID string, email string, userType string) (string, error)

	VerifyAccessToken(tokenStr string) (*Claims, error)
	VerifyRefreshToken(tokenStr string) (*Claims, error)

	GenRandomID() string
	GenVerificationCode() string
}

type auth struct {
	accessSecret  string
	refreshSecret string
}

func NewAuth(accessSecret string, refreshSecret string) Auth {
	return &auth{

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
	log.Println()
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

	return token.SignedString([]byte(a.accessSecret))
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

	return token.SignedString([]byte(a.refreshSecret))
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
		return []byte(a.accessSecret), nil
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
		return []byte(a.refreshSecret), nil
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
//         UTILITY FUNCTIONS       //
//=================================//

func (a *auth) GenRandomID() string {
	randStr := make([]byte, 16)
	io.ReadFull(rand.Reader, randStr)

	return hex.EncodeToString(randStr)
}

func (a *auth) GenVerificationCode() string {
	buf := make([]byte, 6)
	io.ReadFull(rand.Reader, buf)

	return hex.EncodeToString(buf)
}

func (a *auth) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))

	return hex.EncodeToString(hash[:])
}
