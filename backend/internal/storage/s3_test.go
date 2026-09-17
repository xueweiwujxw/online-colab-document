package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestBucketInitializationRetriesFailure(t *testing.T) {
	var requests atomic.Int32
	var allowed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if !allowed.Load() {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	s, err := NewS3Storage(S3Config{Endpoint: server.URL, AccessKey: "test", SecretKey: "test-secret", Bucket: "test-bucket"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ensureBucket(context.Background()); err == nil {
		t.Fatal("expected initial error")
	}
	allowed.Store(true)
	if err := s.ensureBucket(context.Background()); err != nil {
		t.Fatalf("retry: %v", err)
	}
	n := requests.Load()
	if err := s.ensureBucket(context.Background()); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != n {
		t.Fatal("successful initialization should be cached")
	}
}

func TestBucketInitializationConcurrentCreation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(404)
			return
		}
		if r.Method == http.MethodPut {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(409)
			_, _ = w.Write([]byte(`<Error><Code>BucketAlreadyOwnedByYou</Code></Error>`))
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<LocationConstraint>us-east-1</LocationConstraint>`))
	}))
	defer server.Close()
	s, err := NewS3Storage(S3Config{Endpoint: server.URL, AccessKey: "test", SecretKey: "test-secret", Bucket: "test-bucket"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ensureBucket(context.Background()); err != nil {
		t.Fatal(err)
	}
}
