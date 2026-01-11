package storage

import (
	"context"
	"time"
)

// Storage defines the interface for file storage operations
type Storage interface {
	// Upload stores a file and returns its metadata
	Upload(ctx context.Context, filename string, data []byte) (*FileInfo, error)

	// Download retrieves a file by its ID
	Download(ctx context.Context, fileID string) ([]byte, error)

	// Delete removes a file by its ID
	Delete(ctx context.Context, fileID string) error

	// GetInfo retrieves file metadata without downloading content
	GetInfo(ctx context.Context, fileID string) (*FileInfo, error)
}

// FileInfo contains metadata about a stored file
type FileInfo struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	ContentHash string    `json:"content_hash"`
	Path        string    `json:"path"`
	UploadedAt  time.Time `json:"uploaded_at,omitempty"`
}
