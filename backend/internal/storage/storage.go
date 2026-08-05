package storage

import (
	"context"
	"io"
	"time"
)

type Storage interface {
	PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	DeleteObject(ctx context.Context, key string) error
	PresignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type ObjectInfo struct {
	Key         string    `json:"key"`
	SizeBytes   int64     `json:"sizeBytes"`
	ContentType string    `json:"contentType"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Usage struct {
	ObjectCount int64 `json:"objectCount"`
	TotalBytes  int64 `json:"totalBytes"`
}

// AdminStorage is intentionally separate from application Storage so ordinary
// document services cannot enumerate all objects.
type AdminStorage interface {
	Usage(ctx context.Context) (Usage, error)
	ListObjects(ctx context.Context, prefix string, limit int) ([]ObjectInfo, error)
}
