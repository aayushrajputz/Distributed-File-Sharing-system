package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	client           *minio.Client
	externalClient   *minio.Client
	bucket           string
	internalEndpoint string
	externalEndpoint string
}

func NewMinioStorage(endpoint, externalEndpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioStorage, error) {
	// Internal client for operations (uses internal endpoint)
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: "us-east-1", // MinIO requires a region
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// External client for presigned URLs (uses external endpoint accessible from browser)
	externalClient, err := minio.New(externalEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create external minio client: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}

		// Set bucket policy to allow public uploads and downloads
		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {"AWS": ["*"]},
					"Action": ["s3:PutObject", "s3:GetObject"],
					"Resource": ["arn:aws:s3:::%s/*"]
				}
			]
		}`, bucket)

		err = client.SetBucketPolicy(ctx, bucket, policy)
		if err != nil {
			// Log warning but don't fail - policy might already be set
			fmt.Printf("Warning: failed to set bucket policy: %v\n", err)
		}
	}

	return &MinioStorage{
		client:           client,
		externalClient:   externalClient,
		bucket:           bucket,
		internalEndpoint: endpoint,
		externalEndpoint: externalEndpoint,
	}, nil
}

func (s *MinioStorage) GeneratePresignedUploadURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	// Use external client to generate presigned URL with correct signature for external endpoint
	url, err := s.externalClient.PresignedPutObject(ctx, s.bucket, objectName, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	urlStr := url.String()
	fmt.Printf("DEBUG: Generated presigned upload URL: %s\n", urlStr)
	fmt.Printf("DEBUG: Bucket: %s, Object: %s, Expiry: %v\n", s.bucket, objectName, expiry)

	return urlStr, nil
}

func (s *MinioStorage) GeneratePresignedDownloadURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	// Use external client to generate presigned URL with correct signature for external endpoint
	url, err := s.externalClient.PresignedGetObject(ctx, s.bucket, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	urlStr := url.String()
	fmt.Printf("DEBUG: Generated presigned download URL: %s\n", urlStr)

	return urlStr, nil
}

func (s *MinioStorage) UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}

func (s *MinioStorage) DeleteFile(ctx context.Context, objectName string) error {
	err := s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (s *MinioStorage) GetFileInfo(ctx context.Context, objectName string) (*minio.ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	return &info, nil
}
