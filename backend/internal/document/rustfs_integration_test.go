package document

import (
	"context"
	"io"
	"online-colab-document/backend/internal/testutil"
	"strings"
	"testing"
	"time"
)

// Metadata and permissions use the existing fixtures; object IO is real RustFS.
func TestRustFSDocumentVersionsAndPermissions(t *testing.T) {
	s := testutil.RustFS(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	repo := newMemoryRepo()
	permissions := newFakePermissionService()
	service := NewService(repo, s, permissions, 1024)
	for _, ext := range []string{"docx", "xlsx", "md"} {
		t.Run(ext, func(t *testing.T) {
			doc, err := service.Upload(ctx, UploadInput{OwnerID: "owner", OriginalFilename: "example." + ext, HeaderMimeType: map[string]string{"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "md": "text/markdown"}[ext], SizeBytes: 5, Reader: strings.NewReader("first")})
			if err != nil {
				t.Fatal(err)
			}
			permissions.view[doc.ID+":viewer"] = true
			permissions.edit[doc.ID+":editor"] = true
			if _, err := service.Download(ctx, "stranger", doc.ID); err == nil {
				t.Fatal("unauthorized download allowed")
			}
			download, err := service.Download(ctx, "viewer", doc.ID)
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(download.Reader)
			download.Reader.Close()
			if err != nil || string(data) != "first" {
				t.Fatal("upload/download mismatch")
			}
			if ext != "md" {
				return
			}
			first := *doc.CurrentVersionID
			if _, err := service.SaveMarkdown(ctx, "viewer", doc.ID, "forbidden"); err == nil {
				t.Fatal("viewer write allowed")
			}
			if _, err := service.SaveMarkdown(ctx, "editor", doc.ID, "second"); err != nil {
				t.Fatal(err)
			}
			historical, err := service.DownloadVersion(ctx, "viewer", doc.ID, first)
			if err != nil {
				t.Fatal(err)
			}
			data, err = io.ReadAll(historical.Reader)
			historical.Reader.Close()
			if err != nil || string(data) != "first" {
				t.Fatal("historical version changed")
			}
			if _, err := service.RestoreVersion(ctx, "editor", doc.ID, first); err != nil {
				t.Fatal(err)
			}
			download, err = service.Download(ctx, "viewer", doc.ID)
			if err != nil {
				t.Fatal(err)
			}
			data, err = io.ReadAll(download.Reader)
			download.Reader.Close()
			if err != nil || string(data) != "first" {
				t.Fatal("restore failed")
			}
		})
	}
}
