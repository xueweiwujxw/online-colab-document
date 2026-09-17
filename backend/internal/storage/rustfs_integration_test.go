package storage_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"online-colab-document/backend/internal/testutil"
	"testing"
	"time"
)

func TestRustFSObjectLifecycle(t *testing.T) {
	s := testutil.RustFS(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// Unknown length forces the SDK's streaming/multipart upload path.
	cases := []struct {
		key  string
		data []byte
		size int64
	}{
		{"documents/中文 空格+#.md", []byte("# RustFS\n你好"), int64(len("# RustFS\n你好"))},
		{"avatars/empty", nil, 0},
		{"documents/multipart.xlsx", bytes.Repeat([]byte("0123456789abcdef"), 700000), -1},
	}
	var total int64
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			if err := s.PutObject(ctx, tc.key, bytes.NewReader(tc.data), tc.size, "application/octet-stream"); err != nil {
				t.Fatal(err)
			}
			total += int64(len(tc.data))
			r, err := s.GetObject(ctx, tc.key)
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data, tc.data) {
				t.Fatal("download bytes differ")
			}
			signed, err := s.PresignedGetURL(ctx, tc.key, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, signed, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			data, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 200 || !bytes.Equal(data, tc.data) {
				t.Fatalf("presigned download failed: status %d", resp.StatusCode)
			}
			if resp.Header.Get("Content-Type") != "application/octet-stream" {
				t.Fatal("content type lost")
			}
		})
	}
	usage, err := s.Usage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if usage.ObjectCount != 3 || usage.TotalBytes != total {
		t.Fatalf("incorrect usage: %+v", usage)
	}
	objects, err := s.ListObjects(ctx, "/documents/", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 1 {
		t.Fatal("list limit ignored")
	}
	objects, err = s.ListObjects(ctx, "documents/", 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 2 {
		t.Fatal("prefix listing incorrect")
	}
	if err := s.PutObject(ctx, cases[0].key, bytes.NewReader([]byte("updated")), 7, "text/markdown"); err != nil {
		t.Fatal(err)
	}
	r, err := s.GetObject(ctx, cases[0].key)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(r)
	r.Close()
	if err != nil || string(data) != "updated" {
		t.Fatal("overwrite failed")
	}
	for _, tc := range cases {
		if err := s.DeleteObject(ctx, tc.key); err != nil {
			t.Fatal(err)
		}
	}
	r, err = s.GetObject(ctx, cases[0].key)
	if err == nil {
		_, err = io.ReadAll(r)
		r.Close()
	}
	if err == nil {
		t.Fatal("deleted object remains readable")
	}
	usage, err = s.Usage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if usage.ObjectCount != 0 {
		t.Fatal("objects remain after deletion")
	}
}
