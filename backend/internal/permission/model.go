package permission

import "time"

const (
	SubjectTypeUser = "user"

	RoleOwner  = "owner"
	RoleEditor = "editor"
	RoleViewer = "viewer"
	RoleNone   = "none"
)

type Permission struct {
	ID          string
	DocumentID  string
	SubjectType string
	SubjectID   string
	Permission  string
	CreatedBy   *string
	CreatedAt   time.Time
}

type PublicPermission struct {
	ID          string    `json:"id"`
	DocumentID  string    `json:"documentId"`
	SubjectType string    `json:"subjectType"`
	SubjectID   string    `json:"subjectId"`
	Permission  string    `json:"permission"`
	CreatedAt   time.Time `json:"createdAt"`
}

func ToPublic(p Permission) PublicPermission {
	return PublicPermission{
		ID:          p.ID,
		DocumentID:  p.DocumentID,
		SubjectType: p.SubjectType,
		SubjectID:   p.SubjectID,
		Permission:  p.Permission,
		CreatedAt:   p.CreatedAt,
	}
}
