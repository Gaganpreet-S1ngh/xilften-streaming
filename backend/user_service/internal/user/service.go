package user

import (
	"context"
	"fmt"
	"strings"

	pkg "github.com/Gaganpreet-S1ngh/xilften-user-service/pkg/auth"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service interface {
	Register(ctx context.Context, userDetails UserRegisterRequest) (uuid.UUID, error)
	Login(ctx context.Context, userDetails UserLoginRequest, deviceInfo any) (UserLoginResponse, error)
	Logout(ctx context.Context, sessionID string, userID string) error
	VerifyCode(ctx context.Context, userID string, verificationCode string) error
	ForgotPassword(ctx context.Context, userEmail string) error
}

type service struct {
	repository Repository
	auth       pkg.Auth
	logger     *zap.Logger
}

func NewService(repository Repository, logger *zap.Logger, auth pkg.Auth) Service {
	return &service{
		repository: repository,
		auth:       auth,
	}
}

//==========================================//
//               USER AUTH                  //
//==========================================//

// Login implements [Service].
func (s *service) Login(ctx context.Context, userDetails UserLoginRequest, deviceInfo any) (UserLoginResponse, error) {
	existingUser, err := s.repository.FindByEmail(ctx, userDetails.Email)
	if err != nil {
		return UserLoginResponse{}, err
	}

	if existingUser == nil {
		return UserLoginResponse{}, fmt.Errorf("User with this email not found. Please register first!")
	}

	// Verify password
	if err := s.auth.VerifyPassword(userDetails.Password, existingUser.Password); err != nil {
		return UserLoginResponse{}, err
	}
	sessionID := s.auth.GenRandomID()
	// Create tokens access for sending and refresh for storing
	accessToken, err := s.auth.GenerateAccessToken(existingUser.ID.String(), sessionID, existingUser.Email, existingUser.UserType)

	if err != nil {
		return UserLoginResponse{}, err
	}

	refreshToken, err := s.auth.GenerateRefreshToken(existingUser.ID.String(), existingUser.Email, existingUser.UserType)

	if err != nil {
		return UserLoginResponse{}, err
	}

	// Store a new user session in the redis (Can add additional device info)
	if err := s.auth.StoreSession(ctx, refreshToken, existingUser.ID.String(), sessionID); err != nil {
		return UserLoginResponse{}, err
	}

	return UserLoginResponse{
		UserID:      existingUser.ID.String(),
		Email:       existingUser.Email,
		AccessToken: accessToken,
	}, nil
}

// Register implements [Service].
func (s *service) Register(ctx context.Context, userDetails UserRegisterRequest) (uuid.UUID, error) {
	// Check if user already exists
	existingUser, err := s.repository.FindByEmail(ctx, userDetails.Email)
	if err != nil {
		return uuid.Nil, err
	}

	if existingUser != nil {
		return uuid.Nil, fmt.Errorf("User with this email already exists! Please login to continue.")
	}

	// Create hashed password
	hashedPassword, err := s.auth.HashPassword(userDetails.Password)
	if err != nil {
		return uuid.Nil, err
	}

	// Generate a verification Code
	code := s.auth.GenVerificationCode()

	newUser := &User{
		Email:    userDetails.Email,
		Password: hashedPassword,
		Phone:    userDetails.Phone,
		Code:     code,
	}

	userID, err := s.repository.Create(ctx, newUser)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

// Logout implements [Service].
func (s *service) Logout(ctx context.Context, sessionID string, userID string) error {
	return s.auth.RevokeSession(ctx, sessionID, userID)
}

// ForgotPassword implements [Service].
func (s *service) ForgotPassword(ctx context.Context, userEmail string) error {
	panic("unimplemented")
}

// VerifyCode implements [Service].
func (s *service) VerifyCode(ctx context.Context, userID string, verificationCode string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("Error parsing uuid : ", err)
	}

	existingUser, err := s.repository.FindByID(ctx, userUUID)

	// Check
	if existingUser.IsVerified {
		return fmt.Errorf("User is already verified!")
	}

	if len(verificationCode) != 6 {
		return fmt.Errorf("Invalid verification code!")
	}

	if strings.ToLower(verificationCode) != existingUser.Code {
		return fmt.Errorf("Verification code does not match!")
	}

	return nil

}

//==========================================//
//         USER OPERATIONS                  //
//==========================================//
