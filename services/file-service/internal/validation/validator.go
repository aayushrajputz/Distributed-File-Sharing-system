package validation

import (
	"errors"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrInvalidFileName     = errors.New("invalid filename")
	ErrFileNameTooLong     = errors.New("filename too long")
	ErrInvalidCharacters   = errors.New("filename contains invalid characters")
	ErrHiddenFile          = errors.New("hidden files not allowed")
	ErrPathTraversal       = errors.New("path traversal attempt detected")
	ErrInvalidObjectID     = errors.New("invalid object ID format")
	ErrInvalidFileSize     = errors.New("invalid file size")
	ErrUnsupportedMimeType = errors.New("unsupported MIME type")
	ErrInvalidPageSize     = errors.New("invalid page size")
	ErrEmptyField          = errors.New("required field is empty")
)

const (
	MaxFileNameLength = 255
)

// SanitizeFileName removes dangerous characters and prevents path traversal
func SanitizeFileName(name string) (string, error) {
	if name == "" {
		return "", ErrInvalidFileName
	}

	// Remove any directory path components (security measure)
	name = filepath.Base(name)

	// Check for dangerous characters
	if strings.ContainsAny(name, "\\/:*?\"<>|") {
		return "", ErrInvalidCharacters
	}

	// Check for hidden files or traversal attempts
	if strings.HasPrefix(name, ".") {
		return "", ErrHiddenFile
	}

	if strings.Contains(name, "..") {
		return "", ErrPathTraversal
	}

	// Check length
	if len(name) > MaxFileNameLength {
		return "", ErrFileNameTooLong
	}

	// Additional validation - must have at least one character besides extension
	if len(strings.TrimSpace(name)) == 0 {
		return "", ErrInvalidFileName
	}

	return name, nil
}

// ValidateObjectID checks if string is valid MongoDB ObjectID
func ValidateObjectID(id string) error {
	if id == "" {
		return ErrEmptyField
	}

	if _, err := primitive.ObjectIDFromHex(id); err != nil {
		return ErrInvalidObjectID
	}

	return nil
}

// ValidateFileSize checks if file size is within acceptable range
func ValidateFileSize(size, minSize, maxSize int64) error {
	if size < minSize || size > maxSize {
		return ErrInvalidFileSize
	}
	return nil
}

// ValidateMimeType checks if MIME type is in the allowed list
func ValidateMimeType(mimeType string, allowedTypes map[string]bool) error {
	if mimeType == "" {
		return ErrUnsupportedMimeType
	}

	if !allowedTypes[mimeType] {
		return ErrUnsupportedMimeType
	}

	return nil
}

// ValidatePagination ensures pagination parameters are within acceptable ranges
func ValidatePagination(page, limit, maxPageSize int32) (int32, int32, error) {
	if page < 1 {
		page = 1
	}

	if limit < 1 {
		return 0, 0, ErrInvalidPageSize
	}

	if limit > maxPageSize {
		return 0, 0, ErrInvalidPageSize
	}

	return page, limit, nil
}

// ValidateEmail performs basic email validation
func ValidateEmail(email string) error {
	if email == "" {
		return ErrEmptyField
	}

	// Basic email validation
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return errors.New("invalid email format")
	}

	return nil
}

// GenerateSafeStoragePath creates a safe storage path preventing traversal
func GenerateSafeStoragePath(userID, fileName string) (string, error) {
	// Validate user ID
	if err := ValidateObjectID(userID); err != nil {
		return "", err
	}

	// Sanitize filename
	safeName, err := SanitizeFileName(fileName)
	if err != nil {
		return "", err
	}

	// Generate safe path with prefix to avoid collisions
	return filepath.Join("users", userID, "files", safeName), nil
}
