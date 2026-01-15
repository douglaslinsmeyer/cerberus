package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// RustFSClient implements Storage interface using MinIO Go client for S3-compatible RustFS
type RustFSClient struct {
	client     *minio.Client
	bucketName string
	endpoint   string
}

// RustFSConfig holds configuration for RustFS client
type RustFSConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	UseSSL          bool
}

// NewRustFSClient creates a new RustFS storage client using S3 API
func NewRustFSClient(endpoint string) *RustFSClient {
	// Default configuration for development
	config := RustFSConfig{
		Endpoint:        endpoint,
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		BucketName:      "cerberus-artifacts",
		UseSSL:          false,
	}

	return NewRustFSClientWithConfig(config)
}

// NewRustFSClientWithConfig creates a new RustFS client with custom configuration
func NewRustFSClientWithConfig(config RustFSConfig) *RustFSClient {
	// Remove http:// or https:// prefix if present
	endpoint := strings.TrimPrefix(config.Endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	// Initialize MinIO client for S3-compatible storage
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		// Log error but don't fail - will fail on first operation
		fmt.Printf("Warning: Failed to initialize S3 client: %v\n", err)
	}

	rustfs := &RustFSClient{
		client:     client,
		bucketName: config.BucketName,
		endpoint:   config.Endpoint,
	}

	// Try to create bucket if it doesn't exist (best effort)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rustfs.ensureBucket(ctx)

	return rustfs
}

// ensureBucket creates the bucket if it doesn't exist
func (c *RustFSClient) ensureBucket(ctx context.Context) error {
	// Check if bucket exists
	exists, err := c.client.BucketExists(ctx, c.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		// Create bucket
		err = c.client.MakeBucket(ctx, c.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		fmt.Printf("Created bucket: %s\n", c.bucketName)
	}

	return nil
}

// Upload stores a file in RustFS using S3 API
func (c *RustFSClient) Upload(ctx context.Context, filename string, data []byte) (*FileInfo, error) {
	// Generate unique file ID
	fileID := uuid.New().String()

	// Determine content type from file extension
	contentType := "application/octet-stream"
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		contentType = "application/pdf"
	case ".txt":
		contentType = "text/plain"
	case ".json":
		contentType = "application/json"
	case ".csv":
		contentType = "text/csv"
	case ".xlsx":
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".xls":
		contentType = "application/vnd.ms-excel"
	}

	// Create reader from data
	reader := bytes.NewReader(data)
	fileSize := int64(len(data))

	// Upload to S3-compatible storage
	uploadInfo, err := c.client.PutObject(
		ctx,
		c.bucketName,
		fileID,
		reader,
		fileSize,
		minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"original-filename": filename,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Return file info
	return &FileInfo{
		ID:          fileID,
		Filename:    filename,
		Path:        fmt.Sprintf("artifacts/%s", fileID),
		Size:        fileSize,
		ContentHash: uploadInfo.ETag,
		UploadedAt:  time.Now(),
	}, nil
}

// Download retrieves a file from RustFS using S3 API
func (c *RustFSClient) Download(ctx context.Context, fileID string) ([]byte, error) {
	// Get object from S3
	object, err := c.client.GetObject(ctx, c.bucketName, fileID, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	// Read all data
	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	return data, nil
}

// Delete removes a file from RustFS using S3 API
func (c *RustFSClient) Delete(ctx context.Context, fileID string) error {
	err := c.client.RemoveObject(ctx, c.bucketName, fileID, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// GetInfo retrieves file metadata from RustFS using S3 API
func (c *RustFSClient) GetInfo(ctx context.Context, fileID string) (*FileInfo, error) {
	// Get object stat
	stat, err := c.client.StatObject(ctx, c.bucketName, fileID, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	// Extract original filename from metadata
	filename := stat.UserMetadata["Original-Filename"]
	if filename == "" {
		filename = fileID
	}

	return &FileInfo{
		ID:          fileID,
		Filename:    filename,
		Path:        fmt.Sprintf("artifacts/%s", fileID),
		Size:        stat.Size,
		ContentHash: stat.ETag,
		UploadedAt:  stat.LastModified,
	}, nil
}
