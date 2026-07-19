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
	ID                 string
	DocumentID         string
	SubjectType        string
	SubjectID          string
	SubjectDisplayName *string
	SubjectEmail       *string
	Permission         string
	CreatedBy          *string
	CreatedAt          time.Time
}

type PublicPermission struct {
	ID                 string    `json:"id"`
	DocumentID         string    `json:"documentId"`
	SubjectType        string    `json:"subjectType"`
	SubjectID          string    `json:"subjectId"`
	SubjectDisplayName *string   `json:"subjectDisplayName,omitempty"`
	SubjectEmail       *string   `json:"subjectEmail,omitempty"`
	Permission         string    `json:"permission"`
	CreatedAt          time.Time `json:"createdAt"`
}

func ToPublic(p Permission) PublicPermission {
	return PublicPermission{
		ID:                 p.ID,
		DocumentID:         p.DocumentID,
		SubjectType:        p.SubjectType,
		SubjectID:          p.SubjectID,
		SubjectDisplayName: p.SubjectDisplayName,
		SubjectEmail:       p.SubjectEmail,
		Permission:         p.Permission,
		CreatedAt:          p.CreatedAt,
	}
}
