package permission

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid permission input")
)

type Service struct {
	repo  Repository
	now   func() time.Time
	newID func() (string, error)
}

type GrantInput struct {
	ActorID     string
	DocumentID  string
	SubjectType string
	SubjectID   string
	Permission  string
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:  repo,
		now:   time.Now,
		newID: newUUID,
	}
}

func (s *Service) Role(ctx context.Context, userID string, documentID string) (string, error) {
	if userID == "" || documentID == "" {
		return RoleNone, nil
	}
	ownerID, err := s.repo.DocumentOwnerID(ctx, documentID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return RoleNone, nil
		}
		return RoleNone, err
	}
	if ownerID == userID {
		return RoleOwner, nil
	}
	permission, err := s.repo.FindForUser(ctx, documentID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return RoleNone, nil
		}
		return RoleNone, err
	}
	if permission.Permission != RoleEditor && permission.Permission != RoleViewer {
		return RoleNone, nil
	}
	return permission.Permission, nil
}

func (s *Service) CanView(ctx context.Context, userID string, documentID string) (bool, error) {
	role, err := s.Role(ctx, userID, documentID)
	return role == RoleOwner || role == RoleEditor || role == RoleViewer, err
}

func (s *Service) CanEdit(ctx context.Context, userID string, documentID string) (bool, error) {
	role, err := s.Role(ctx, userID, documentID)
	return role == RoleOwner || role == RoleEditor, err
}

func (s *Service) CanManage(ctx context.Context, userID string, documentID string) (bool, error) {
	role, err := s.Role(ctx, userID, documentID)
	return role == RoleOwner, err
}

func (s *Service) CanDelete(ctx context.Context, userID string, documentID string) (bool, error) {
	role, err := s.Role(ctx, userID, documentID)
	return role == RoleOwner, err
}

func (s *Service) CanShare(ctx context.Context, userID string, documentID string) (bool, error) {
	role, err := s.Role(ctx, userID, documentID)
	return role == RoleOwner, err
}

func (s *Service) List(ctx context.Context, actorID string, documentID string) ([]Permission, error) {
	canManage, err := s.CanManage(ctx, actorID, documentID)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, ErrForbidden
	}
	return s.repo.ListForDocument(ctx, documentID)
}

func (s *Service) Grant(ctx context.Context, input GrantInput) (Permission, error) {
	if input.SubjectType != SubjectTypeUser || input.SubjectID == "" || input.DocumentID == "" {
		return Permission{}, ErrInvalidInput
	}
	if input.Permission != RoleEditor && input.Permission != RoleViewer {
		return Permission{}, ErrInvalidInput
	}
	canManage, err := s.CanManage(ctx, input.ActorID, input.DocumentID)
	if err != nil {
		return Permission{}, err
	}
	if !canManage {
		return Permission{}, ErrForbidden
	}
	id, err := s.newID()
	if err != nil {
		return Permission{}, err
	}
	createdBy := input.ActorID
	permission := Permission{
		ID:          id,
		DocumentID:  input.DocumentID,
		SubjectType: input.SubjectType,
		SubjectID:   input.SubjectID,
		Permission:  input.Permission,
		CreatedBy:   &createdBy,
		CreatedAt:   s.now().UTC(),
	}
	if err := s.repo.Create(ctx, permission); err != nil {
		return Permission{}, err
	}
	return permission, nil
}

func (s *Service) Delete(ctx context.Context, actorID string, documentID string, permissionID string) error {
	if permissionID == "" || documentID == "" {
		return ErrInvalidInput
	}
	canManage, err := s.CanManage(ctx, actorID, documentID)
	if err != nil {
		return err
	}
	if !canManage {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, documentID, permissionID)
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
