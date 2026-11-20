package service

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/nhassl3/servicehub-backend/internal/domain"
)

// extractObjectName extracts the object path from a full MinIO URL.
// e.g. "http://localhost:9000/servicehub-avatars/avatars/users/john/abc.jpg"
// returns "avatars/users/john/abc.jpg"
func extractObjectName(url string) string {
	// Find the bucket name + "/" then take everything after it
	parts := strings.SplitN(url, "/", 5) // [http:, "", host, bucket, objectName]
	if len(parts) < 5 {
		return ""
	}
	return parts[4]
}

// ObjectNameFromURL is exported for use by other services.
func ObjectNameFromURL(url string) string {
	return extractObjectName(url)
}

// ValidateFileUpload checks file size and content type.
func ValidateFileUpload(data []byte, contentType string) error {
	if len(data) > maxAvatarSize {
		return domain.ErrFileTooLarge
	}
	if !strings.HasPrefix(contentType, "image/") {
		return fmt.Errorf("%s: %w", contentType, domain.ErrInvalidFileType)
	}
	return nil
}

// GenerateObjectName creates a unique object path for file storage.
func GenerateObjectName(prefix, owner, contentType string) string {
	ext, err := mime.ExtensionsByType(contentType)
	if err != nil {
		return filepath.Join(prefix, owner, uuid.NewString())
	}
	return filepath.Join(prefix, owner, uuid.NewString()+ext[0])
}
