package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type MinIOStorage struct {
	client     *minio.Client
	bucket     string
	ensureOnce sync.Once
	ensureErr  error
}

func NewMinIOStorage(cfg MinIOConfig) (*MinIOStorage, error) {
	endpoint, useSSL, err := parseEndpoint(cfg.Endpoint, cfg.UseSSL)
	if err != nil {
		return nil, err
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	return &MinIOStorage{client: client, bucket: cfg.Bucket}, nil
}

func (s *MinIOStorage) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	if err := s.ensureBucket(ctx); err != nil {
		return err
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func (s *MinIOStorage) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	return object, nil
}

func (s *MinIOStorage) DeleteObject(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func (s *MinIOStorage) PresignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("presign object: %w", err)
	}
	return u.String(), nil
}

func (s *MinIOStorage) Usage(ctx context.Context) (Usage, error) {
	if err := s.ensureBucket(ctx); err != nil {
		return Usage{}, err
	}
	var usage Usage
	for object := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Recursive: true}) {
		if object.Err != nil {
			return Usage{}, fmt.Errorf("list storage usage: %w", object.Err)
		}
		usage.ObjectCount++
		usage.TotalBytes += object.Size
	}
	return usage, nil
}

func (s *MinIOStorage) ListObjects(ctx context.Context, prefix string, limit int) ([]ObjectInfo, error) {
	if err := s.ensureBucket(ctx); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	items := make([]ObjectInfo, 0, limit)
	for object := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: strings.TrimPrefix(prefix, "/"), Recursive: true}) {
		if object.Err != nil {
			return nil, fmt.Errorf("list storage objects: %w", object.Err)
		}
		items = append(items, ObjectInfo{Key: object.Key, SizeBytes: object.Size, ContentType: object.ContentType, UpdatedAt: object.LastModified})
		if len(items) == limit {
			break
		}
	}
	return items, nil
}

func (s *MinIOStorage) ensureBucket(ctx context.Context) error {
	s.ensureOnce.Do(func() {
		exists, err := s.client.BucketExists(ctx, s.bucket)
		if err != nil {
			s.ensureErr = fmt.Errorf("check storage bucket: %w", err)
			return
		}
		if exists {
			return
		}
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			s.ensureErr = fmt.Errorf("create storage bucket: %w", err)
		}
	})
	return s.ensureErr
}

func parseEndpoint(raw string, fallbackUseSSL bool) (string, bool, error) {
	if raw == "" {
		return "", fallbackUseSSL, fmt.Errorf("storage endpoint is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fallbackUseSSL, fmt.Errorf("parse storage endpoint: %w", err)
	}
	if parsed.Scheme == "" {
		return raw, fallbackUseSSL, nil
	}
	if parsed.Host == "" {
		return "", fallbackUseSSL, fmt.Errorf("storage endpoint host is required")
	}
	useSSL := parsed.Scheme == "https"
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fallbackUseSSL, fmt.Errorf("unsupported storage endpoint scheme %q", parsed.Scheme)
	}
	return strings.TrimRight(parsed.Host, "/"), useSSL, nil
}
