package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/distributed-file-sharing/services/auth-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/auth-service/internal/repository"
	"github.com/yourusername/distributed-file-sharing/services/auth-service/internal/service"
	authv1 "github.com/yourusername/distributed-file-sharing/services/auth-service/pkg/pb/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	userRepo        *repository.UserRepository
	jwtService      *service.JWTService
	passwordService *service.PasswordService
}

func NewAuthHandler(
	userRepo *repository.UserRepository,
	jwtService *service.JWTService,
	passwordService *service.PasswordService,
) *AuthHandler {
	return &AuthHandler{
		userRepo:        userRepo,
		jwtService:      jwtService,
		passwordService: passwordService,
	}
}

func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	// Validate input
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		return nil, status.Error(codes.InvalidArgument, "email, password, and full_name are required")
	}

	// Hash password
	hashedPassword, err := h.passwordService.HashPassword(req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	// Create user
	user := &models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FullName:     req.FullName,
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
		}
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	return &authv1.RegisterResponse{
		User: &authv1.User{
			UserId:    user.ID.Hex(),
			Email:     user.Email,
			FullName:  user.FullName,
			AvatarUrl: user.AvatarURL,
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
		},
		Message: "User registered successfully",
	}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	// Validate input
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	// Find user
	user, err := h.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		return nil, status.Error(codes.Internal, "failed to find user")
	}

	// Check password
	if !h.passwordService.CheckPassword(req.Password, user.PasswordHash) {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	// Generate tokens
	accessToken, expiresIn, err := h.jwtService.GenerateAccessToken(user.ID.Hex(), user.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(user.ID.Hex(), user.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate refresh token")
	}

	return &authv1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		User: &authv1.User{
			UserId:    user.ID.Hex(),
			Email:     user.Email,
			FullName:  user.FullName,
			AvatarUrl: user.AvatarURL,
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
		},
	}, nil
}

func (h *AuthHandler) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	claims, err := h.jwtService.ValidateToken(req.Token)
	if err != nil {
		if errors.Is(err, service.ErrExpiredToken) {
			return &authv1.ValidateTokenResponse{
				Valid:   false,
				Message: "token has expired",
			}, nil
		}
		return &authv1.ValidateTokenResponse{
			Valid:   false,
			Message: "invalid token",
		}, nil
	}

	return &authv1.ValidateTokenResponse{
		Valid:  true,
		UserId: claims.UserID,
		Email:  claims.Email,
	}, nil
}

func (h *AuthHandler) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	user, err := h.userRepo.FindByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to find user")
	}

	return &authv1.GetUserResponse{
		User: &authv1.User{
			UserId:    user.ID.Hex(),
			Email:     user.Email,
			FullName:  user.FullName,
			AvatarUrl: user.AvatarURL,
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
		},
	}, nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	claims, err := h.jwtService.ValidateToken(req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}

	// Generate new access token
	accessToken, expiresIn, err := h.jwtService.GenerateAccessToken(claims.UserID, claims.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	return &authv1.RefreshTokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
	}, nil
}

func (h *AuthHandler) UpdateProfile(ctx context.Context, req *authv1.UpdateProfileRequest) (*authv1.UpdateProfileResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	user, err := h.userRepo.FindByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to find user")
	}

	// Update fields
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.AvatarUrl != "" {
		user.AvatarURL = req.AvatarUrl
	}

	if err := h.userRepo.Update(ctx, user); err != nil {
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	return &authv1.UpdateProfileResponse{
		User: &authv1.User{
			UserId:    user.ID.Hex(),
			Email:     user.Email,
			FullName:  user.FullName,
			AvatarUrl: user.AvatarURL,
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
		},
		Message: "Profile updated successfully",
	}, nil
}

func (h *AuthHandler) ChangePassword(ctx context.Context, req *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error) {
	// Validate input
	if req.UserId == "" || req.CurrentPassword == "" || req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, current_password, and new_password are required")
	}

	// Validate new password length
	if len(req.NewPassword) < 8 {
		return nil, status.Error(codes.InvalidArgument, "new password must be at least 8 characters long")
	}

	// Find user
	user, err := h.userRepo.FindByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to find user")
	}

	// Verify current password
	if !h.passwordService.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		return nil, status.Error(codes.Unauthenticated, "current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := h.passwordService.HashPassword(req.NewPassword)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash new password")
	}

	// Update password
	user.PasswordHash = hashedPassword
	user.UpdatedAt = time.Now()

	if err := h.userRepo.UpdatePassword(ctx, user); err != nil {
		return nil, status.Error(codes.Internal, "failed to update password")
	}

	return &authv1.ChangePasswordResponse{
		Message: "Password changed successfully",
	}, nil
}
