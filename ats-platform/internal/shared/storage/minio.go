package storage

import (
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

// FileStorage defines the interface for file storage operations
type FileStorage interface {
	UploadFile(ctx context.Context, reader io.Reader, objectName string, contentType string, size int64) (string, error)
	GetFileURL(objectKey string) string
	GetPresignedURL(ctx context.Context, objectKey string) (string, error)
	DeleteFile(ctx context.Context, objectKey string) error
	EnsureBucket(ctx context.Context) error
	DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error)
}

// MinIOConfig holds the configuration for MinIO client
type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
}

// MinIOClient wraps the MinIO client with custom functionality
type MinIOClient struct {
	client   *minio.Client
	bucket   string
	endpoint string
}

// NewMinIOClient creates a new MinIO client
func NewMinIOClient(cfg MinIOConfig) (*MinIOClient, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinIOClient{
		client:   client,
		bucket:   cfg.Bucket,
		endpoint: cfg.Endpoint,
	}, nil
}

// EnsureBucket creates the bucket if it doesn't exist
func (m *MinIOClient) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

// UploadFile uploads a file to MinIO and returns the object key
func (m *MinIOClient) UploadFile(ctx context.Context, reader io.Reader, objectName string, contentType string, size int64) (string, error) {
	// Generate unique object name with folder structure
	ext := filepath.Ext(objectName)
	objectKey := fmt.Sprintf("resumes/%s/%s%s", time.Now().Format("2006/01/02"), uuid.New().String(), ext)

	_, err := m.client.PutObject(ctx, m.bucket, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return objectKey, nil
}

// GetFileURL returns the URL for accessing a file
func (m *MinIOClient) GetFileURL(objectKey string) string {
	// Return HTTP URL for the object (always http for local dev)
	return fmt.Sprintf("http://%s/%s/%s", m.endpoint, m.bucket, objectKey)
}

// GetPresignedURL returns a presigned URL for temporary access (valid for 1 hour)
func (m *MinIOClient) GetPresignedURL(ctx context.Context, objectKey string) (string, error) {
	url, err := m.client.PresignedGetObject(ctx, m.bucket, objectKey, time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

// DeleteFile deletes a file from MinIO
func (m *MinIOClient) DeleteFile(ctx context.Context, objectKey string) error {
	err := m.client.RemoveObject(ctx, m.bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// DownloadFile downloads a file from MinIO
func (m *MinIOClient) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	object, err := m.client.GetObject(ctx, m.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	return object, nil
}

// IsAllowedFileType checks if the file type is allowed
func IsAllowedFileType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	allowedTypes := map[string]bool{
		".pdf":  true,
		".doc":  true,
		".docx": true,
	}
	return allowedTypes[ext]
}

// GetContentType returns the content type based on file extension
func GetContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	contentTypes := map[string]string{
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}
	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}
