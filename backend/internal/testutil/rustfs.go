// Package testutil provides opt-in integration fixtures; it never uses application credentials.
package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"net/url"
	"online-colab-document/backend/internal/storage"
)

// RustFS creates an isolated bucket and removes only that bucket on cleanup.
func RustFS(t *testing.T) *storage.S3Storage {
	t.Helper()
	endpoint := os.Getenv("RUSTFS_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("set RUSTFS_TEST_ENDPOINT to run real RustFS integration tests")
	}
	access, secret := os.Getenv("RUSTFS_TEST_ACCESS_KEY"), os.Getenv("RUSTFS_TEST_SECRET_KEY")
	if access == "" || secret == "" {
		t.Fatal("RUSTFS_TEST_ACCESS_KEY and RUSTFS_TEST_SECRET_KEY are required")
	}
	bucket := fmt.Sprintf("test-%d", time.Now().UnixNano())
	s, err := storage.NewS3Storage(storage.S3Config{Endpoint: endpoint, AccessKey: access, SecretKey: secret, Bucket: bucket})
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	client, err := minio.New(u.Host, &minio.Options{Creds: credentials.NewStaticV4(access, secret, ""), Secure: u.Scheme == "https", BucketLookup: minio.BucketLookupPath})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		exists, err := client.BucketExists(ctx, bucket)
		if err != nil {
			t.Error(err)
			return
		}
		if !exists {
			return
		}
		for obj := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true}) {
			if obj.Err != nil {
				t.Error(obj.Err)
				return
			}
			if err := client.RemoveObject(ctx, bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
				t.Error(err)
			}
		}
		if err := client.RemoveBucket(ctx, bucket); err != nil {
			t.Error(err)
		}
	})
	return s
}
