package share

import (
	"time"

	"online-colab-document/backend/internal/document"
)

const (
	PermissionViewer = "viewer"
	PermissionEditor = "editor"
)

type Link struct {
	ID         string
	DocumentID string
	TokenHash  string
	Permission string
	ExpiresAt  *time.Time
	Disabled   bool
	CreatedBy  *string
	CreatedAt  time.Time
}

type PublicLink struct {
	ID         string     `json:"id"`
	DocumentID string     `json:"documentId"`
	Permission string     `json:"permission"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	Disabled   bool       `json:"disabled"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type CreateResponse struct {
	PublicLink
	Token string `json:"token"`
	URL   string `json:"url"`
}

type AccessResponse struct {
	Document document.PublicDocument `json:"document"`
	Content  string                  `json:"content,omitempty"`
	CanEdit  bool                    `json:"canEdit"`
}

func ToPublic(link Link) PublicLink {
	return PublicLink{
		ID:         link.ID,
		DocumentID: link.DocumentID,
		Permission: link.Permission,
		ExpiresAt:  link.ExpiresAt,
		Disabled:   link.Disabled,
		CreatedAt:  link.CreatedAt,
	}
}
