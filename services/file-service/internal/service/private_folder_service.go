package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/models"
	"github.com/yourusername/distributed-file-sharing/services/file-service/internal/repository"
)

// PrivateFolderService handles private folder business logic
type PrivateFolderService struct {
	pinRepo     *repository.PrivateFolderRepository
	fileRepo    *repository.FileRepository
	storageRepo *repository.StorageRepository
}

// NewPrivateFolderService creates a new private folder service
func NewPrivateFolderService(
	pinRepo *repository.PrivateFolderRepository,
	fileRepo *repository.FileRepository,
	storageRepo *repository.StorageRepository,
) *PrivateFolderService {
	return &PrivateFolderService{
		pinRepo:     pinRepo,
		fileRepo:    fileRepo,
		storageRepo: storageRepo,
	}
}

// SetPIN sets or updates a user's PIN
func (s *PrivateFolderService) SetPIN(ctx context.Context, userID, pin string) error {
	// Validate PIN
	if len(pin) < models.PINLength || len(pin) > models.MaxPINLength {
		return fmt.Errorf("PIN must be between %d and %d characters", models.PINLength, models.MaxPINLength)
	}

	// Generate salt
	salt := generateSalt()

	// Hash PIN with salt
	hashedPIN, err := bcrypt.GenerateFromPassword([]byte(pin+salt), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash PIN: %w", err)
	}

	// Store in database
	return s.pinRepo.CreateOrUpdatePIN(ctx, userID, string(hashedPIN), salt)
}

// ValidatePIN validates a user's PIN with brute force protection
func (s *PrivateFolderService) ValidatePIN(ctx context.Context, req *models.PINValidationRequest) (*models.PINValidationResponse, error) {
	// Check if user is locked out
	pin, err := s.pinRepo.GetPIN(ctx, req.UserID)
	if err != nil {
		return &models.PINValidationResponse{
			Success: false,
			Message: "PIN not set. Please set a PIN first.",
		}, nil
	}

	// Check if PIN is locked
	if pin.LockedUntil != nil && time.Now().Before(*pin.LockedUntil) {
		return &models.PINValidationResponse{
			Success:     false,
			Message:     "Account locked due to too many failed attempts",
			LockedUntil: pin.LockedUntil.Format(time.RFC3339),
		}, nil
	}

	// Check IP-based attempts
	attempts, err := s.pinRepo.GetPINAttempts(ctx, req.UserID, req.IPAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to check PIN attempts: %w", err)
	}

	if attempts != nil && attempts.IsBlocked && attempts.BlockedUntil != nil && time.Now().Before(*attempts.BlockedUntil) {
		return &models.PINValidationResponse{
			Success:     false,
			Message:     "IP address blocked due to too many failed attempts",
			LockedUntil: attempts.BlockedUntil.Format(time.RFC3339),
		}, nil
	}

	// Validate PIN
	err = bcrypt.CompareHashAndPassword([]byte(pin.PINHash), []byte(req.PIN+pin.Salt))
	if err != nil {
		// Log failed attempt
		s.logAccess(ctx, req.UserID, "", models.ActionPINFailed, req.IPAddress, req.UserAgent, false, "Invalid PIN")

		// Update failed attempts
		newAttempts := pin.FailedAttempts + 1
		var lockedUntil *time.Time
		var attemptsLeft int

		if newAttempts >= models.MaxPINAttempts {
			lockDuration := models.PINLockoutDuration
			lockTime := time.Now().Add(lockDuration)
			lockedUntil = &lockTime
			attemptsLeft = 0
		} else {
			attemptsLeft = models.MaxPINAttempts - newAttempts
		}

		// Update database
		s.pinRepo.UpdateFailedAttempts(ctx, req.UserID, newAttempts, lockedUntil)
		s.pinRepo.UpdatePINAttempts(ctx, req.UserID, req.IPAddress, newAttempts >= models.MaxPINAttempts, lockedUntil)

		return &models.PINValidationResponse{
			Success:      false,
			Message:      "Invalid PIN",
			AttemptsLeft: attemptsLeft,
			LockedUntil:  lockedUntil.Format(time.RFC3339),
		}, nil
	}

	// PIN is valid - reset failed attempts
	s.pinRepo.ResetFailedAttempts(ctx, req.UserID)
	s.pinRepo.ResetPINAttempts(ctx, req.UserID, req.IPAddress)

	// Log successful attempt
	s.logAccess(ctx, req.UserID, "", models.ActionPINVerified, req.IPAddress, req.UserAgent, true, "")

	return &models.PINValidationResponse{
		Success: true,
		Message: "PIN validated successfully",
	}, nil
}

// MakeFilePrivate moves a file to private folder
func (s *PrivateFolderService) MakeFilePrivate(ctx context.Context, req *models.MakePrivateRequest) (*models.MakePrivateResponse, error) {
	// First validate PIN
	pinReq := &models.PINValidationRequest{
		UserID: req.UserID,
		PIN:    req.PIN,
	}

	pinResp, err := s.ValidatePIN(ctx, pinReq)
	if err != nil {
		return nil, fmt.Errorf("failed to validate PIN: %w", err)
	}

	if !pinResp.Success {
		return &models.MakePrivateResponse{
			Success: false,
			Message: pinResp.Message,
		}, nil
	}

	// Get file information
	file, err := s.fileRepo.FindByID(ctx, req.FileID)
	if err != nil {
		return &models.MakePrivateResponse{
			Success: false,
			Message: "File not found",
		}, nil
	}

	// Check if user owns the file
	if file.OwnerID != req.UserID {
		return &models.MakePrivateResponse{
			Success: false,
			Message: "Access denied",
		}, nil
	}

	// Check if file is already private
	isPrivate, err := s.pinRepo.IsFilePrivate(ctx, req.UserID, req.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to check file privacy status: %w", err)
	}

	if isPrivate {
		return &models.MakePrivateResponse{
			Success: false,
			Message: "File is already private",
		}, nil
	}

	// Add file to private folder
	err = s.pinRepo.AddFileToPrivateFolder(ctx, req.UserID, req.FileID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to add file to private folder: %w", err)
	}

	// Update file metadata
	file.IsPrivate = true
	err = s.fileRepo.Update(ctx, file)
	if err != nil {
		// Rollback private folder addition
		s.pinRepo.RemoveFileFromPrivateFolder(ctx, req.UserID, req.FileID)
		return nil, fmt.Errorf("failed to update file metadata: %w", err)
	}

	// Log the action
	s.logAccess(ctx, req.UserID, req.FileID, models.ActionFileMovedToPrivate, "", "", true, "")

	return &models.MakePrivateResponse{
		Success: true,
		Message: "File moved to private folder successfully",
		FileID:  req.FileID,
	}, nil
}

// RemoveFileFromPrivate moves a file out of private folder
func (s *PrivateFolderService) RemoveFileFromPrivate(ctx context.Context, userID, fileID, pin string) (*models.MakePrivateResponse, error) {
	// Validate PIN
	pinReq := &models.PINValidationRequest{
		UserID: userID,
		PIN:    pin,
	}

	pinResp, err := s.ValidatePIN(ctx, pinReq)
	if err != nil {
		return nil, fmt.Errorf("failed to validate PIN: %w", err)
	}

	if !pinResp.Success {
		return &models.MakePrivateResponse{
			Success: false,
			Message: pinResp.Message,
		}, nil
	}

	// Remove file from private folder
	err = s.pinRepo.RemoveFileFromPrivateFolder(ctx, userID, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to remove file from private folder: %w", err)
	}

	// Update file metadata
	file, err := s.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to find file: %w", err)
	}

	file.IsPrivate = false
	err = s.fileRepo.Update(ctx, file)
	if err != nil {
		return nil, fmt.Errorf("failed to update file metadata: %w", err)
	}

	// Log the action
	s.logAccess(ctx, userID, fileID, models.ActionFileMovedFromPrivate, "", "", true, "")

	return &models.MakePrivateResponse{
		Success: true,
		Message: "File removed from private folder successfully",
		FileID:  fileID,
	}, nil
}

// GetPrivateFiles retrieves all private files for a user
func (s *PrivateFolderService) GetPrivateFiles(ctx context.Context, userID string, limit, offset int64) (*models.PrivateFolderListResponse, error) {
	files, total, err := s.pinRepo.GetPrivateFiles(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get private files: %w", err)
	}

	return &models.PrivateFolderListResponse{
		Files: files,
		Total: total,
	}, nil
}

// GetAccessLogs retrieves access logs for a user
func (s *PrivateFolderService) GetAccessLogs(ctx context.Context, userID string, limit int64) ([]models.PrivateFolderAccessLog, error) {
	return s.pinRepo.GetAccessLogs(ctx, userID, limit)
}

// CheckFileAccess checks if user can access a private file
func (s *PrivateFolderService) CheckFileAccess(ctx context.Context, userID, fileID string) (bool, error) {
	// Check if file is private
	isPrivate, err := s.pinRepo.IsFilePrivate(ctx, userID, fileID)
	if err != nil {
		return false, fmt.Errorf("failed to check file privacy: %w", err)
	}

	if !isPrivate {
		return true, nil // File is not private, access allowed
	}

	// File is private - user needs PIN validation
	return false, nil
}

// Helper function to generate salt
func generateSalt() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Helper function to log access
func (s *PrivateFolderService) logAccess(_ context.Context, userID, fileID, action, ipAddress, userAgent string, success bool, failureReason string) {
	log := &models.PrivateFolderAccessLog{
		ID:            primitive.NewObjectID(),
		UserID:        userID,
		FileID:        fileID,
		Action:        action,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		Success:       success,
		FailureReason: failureReason,
		CreatedAt:     time.Now(),
	}

	// Log asynchronously to avoid blocking the main operation
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.pinRepo.LogAccess(ctx, log)
	}()
}

