package permission

import (
	"context"
	"testing"
	"time"
)

func TestPermissionMatrix(t *testing.T) {
	repo := newMemoryRepo()
	repo.owners["doc-1"] = "owner-1"
	repo.permissions["doc-1:editor-1"] = Permission{DocumentID: "doc-1", SubjectType: SubjectTypeUser, SubjectID: "editor-1", Permission: RoleEditor}
	repo.permissions["doc-1:viewer-1"] = Permission{DocumentID: "doc-1", SubjectType: SubjectTypeUser, SubjectID: "viewer-1", Permission: RoleViewer}
	service := NewService(repo)

	cases := []struct {
		name      string
		userID    string
		canView   bool
		canEdit   bool
		canManage bool
		canDelete bool
	}{
		{name: "owner", userID: "owner-1", canView: true, canEdit: true, canManage: true, canDelete: true},
		{name: "editor", userID: "editor-1", canView: true, canEdit: true, canManage: false, canDelete: false},
		{name: "viewer", userID: "viewer-1", canView: true, canEdit: false, canManage: false, canDelete: false},
		{name: "none", userID: "other-1", canView: false, canEdit: false, canManage: false, canDelete: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := service.CanView(context.Background(), tc.userID, "doc-1")
			assertPermission(t, "CanView", actual, err, tc.canView)
			actual, err = service.CanEdit(context.Background(), tc.userID, "doc-1")
			assertPermission(t, "CanEdit", actual, err, tc.canEdit)
			actual, err = service.CanManage(context.Background(), tc.userID, "doc-1")
			assertPermission(t, "CanManage", actual, err, tc.canManage)
			actual, err = service.CanDelete(context.Background(), tc.userID, "doc-1")
			assertPermission(t, "CanDelete", actual, err, tc.canDelete)
		})
	}
}

func TestGrantAndDeletePermissionTakeEffectImmediately(t *testing.T) {
	repo := newMemoryRepo()
	repo.owners["doc-1"] = "owner-1"
	service := NewService(repo)

	_, err := service.Grant(context.Background(), GrantInput{
		ActorID:     "owner-1",
		DocumentID:  "doc-1",
		SubjectType: SubjectTypeUser,
		SubjectID:   "viewer-1",
		Permission:  RoleViewer,
	})
	if err != nil {
		t.Fatalf("grant permission: %v", err)
	}
	actual, err := service.CanView(context.Background(), "viewer-1", "doc-1")
	assertPermission(t, "CanView after grant", actual, err, true)
	actual, err = service.CanEdit(context.Background(), "viewer-1", "doc-1")
	assertPermission(t, "CanEdit after grant", actual, err, false)

	var permissionID string
	for _, permission := range repo.permissions {
		permissionID = permission.ID
	}
	if err := service.Delete(context.Background(), "owner-1", "doc-1", permissionID); err != nil {
		t.Fatalf("delete permission: %v", err)
	}
	actual, err = service.CanView(context.Background(), "viewer-1", "doc-1")
	assertPermission(t, "CanView after delete", actual, err, false)
}

func TestEditorCannotManagePermissions(t *testing.T) {
	repo := newMemoryRepo()
	repo.owners["doc-1"] = "owner-1"
	repo.permissions["doc-1:editor-1"] = Permission{ID: "permission-1", DocumentID: "doc-1", SubjectType: SubjectTypeUser, SubjectID: "editor-1", Permission: RoleEditor}
	service := NewService(repo)

	_, err := service.Grant(context.Background(), GrantInput{
		ActorID:     "editor-1",
		DocumentID:  "doc-1",
		SubjectType: SubjectTypeUser,
		SubjectID:   "viewer-1",
		Permission:  RoleViewer,
	})
	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if err := service.Delete(context.Background(), "editor-1", "doc-1", "permission-1"); err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func assertPermission(t *testing.T, label string, actual bool, err error, expected bool) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", label, err)
	}
	if actual != expected {
		t.Fatalf("%s expected %v, got %v", label, expected, actual)
	}
}

type memoryRepo struct {
	owners      map[string]string
	permissions map[string]Permission
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		owners:      map[string]string{},
		permissions: map[string]Permission{},
	}
}

func (r *memoryRepo) DocumentOwnerID(_ context.Context, documentID string) (string, error) {
	ownerID, ok := r.owners[documentID]
	if !ok {
		return "", ErrNotFound
	}
	return ownerID, nil
}

func (r *memoryRepo) FindForUser(_ context.Context, documentID string, userID string) (Permission, error) {
	permission, ok := r.permissions[documentID+":"+userID]
	if !ok {
		return Permission{}, ErrNotFound
	}
	return permission, nil
}

func (r *memoryRepo) ListForDocument(_ context.Context, documentID string) ([]Permission, error) {
	var permissions []Permission
	for _, permission := range r.permissions {
		if permission.DocumentID == documentID {
			permissions = append(permissions, permission)
		}
	}
	return permissions, nil
}

func (r *memoryRepo) Create(_ context.Context, permission Permission) error {
	if permission.ID == "" {
		permission.ID = "permission-" + permission.SubjectID
	}
	if permission.CreatedAt.IsZero() {
		permission.CreatedAt = time.Now()
	}
	r.permissions[permission.DocumentID+":"+permission.SubjectID] = permission
	return nil
}

func (r *memoryRepo) Delete(_ context.Context, documentID string, permissionID string) error {
	for key, permission := range r.permissions {
		if permission.DocumentID == documentID && permission.ID == permissionID {
			delete(r.permissions, key)
			return nil
		}
	}
	return ErrNotFound
}
