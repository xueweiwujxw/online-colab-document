package collab

import "time"

type Snapshot struct {
	ID         string
	DocumentID string
	VersionNo  int64
	Content    string
	CreatedBy  *string
	CreatedAt  time.Time
}

type Update struct {
	ID         string
	DocumentID string
	UpdateSeq  int64
	UpdateData []byte
	CreatedBy  *string
	CreatedAt  time.Time
}

type PublicSnapshot struct {
	DocumentID string         `json:"documentId"`
	Content    string         `json:"content"`
	VersionNo  int64          `json:"versionNo"`
	CanEdit    bool           `json:"canEdit"`
	Users      []PresenceUser `json:"users"`
}

type PresenceUser struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	CanEdit     bool   `json:"canEdit"`
}
