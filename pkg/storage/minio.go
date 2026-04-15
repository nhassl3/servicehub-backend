package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nhassl3/servicehub-backend/internal/config"
)

type MinIO struct {
	client *minio.Client
	cfg    *config.MinIOConfig
}

func NewMinIO(ctx context.Context, cfg *config.MinIOConfig) (*MinIO, error) {
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:      credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:     cfg.UseSSL,
		MaxRetries: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: create client: %w", err)
	}

	exists, err := minioClient.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("minio: check bucket %q: %w", cfg.Bucket, err)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio: create bucket %q: %w", cfg.Bucket, err)
		}

		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}]
		}`, cfg.Bucket)
		if err := minioClient.SetBucketPolicy(ctx, cfg.Bucket, policy); err != nil {
			return nil, fmt.Errorf("minio: set bucket policy: %w", err)
		}
	}

	return &MinIO{client: minioClient, cfg: cfg}, nil
}

func (m *MinIO) Upload(
	ctx context.Context,
	objectName, contentType string,
	reader io.Reader,
	size int64,
) (string, error) {
	_, err := m.client.PutObject(ctx, m.cfg.Bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("minio: upload %q: %w", objectName, err)
	}
	return m.objectURL(objectName), nil
}

func (m *MinIO) Delete(ctx context.Context, objectName string) error {
	err := m.client.RemoveObject(ctx, m.cfg.Bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("minio: delete %q: %w", objectName, err)
	}
	return nil
}

func (m *MinIO) objectURL(objectName string) string {
	protocol := "http"
	if m.cfg.UseSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", protocol, m.cfg.Endpoint, m.cfg.Bucket, objectName)
}
