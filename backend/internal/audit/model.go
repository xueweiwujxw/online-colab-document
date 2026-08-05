package audit

import "time"

const (
	ActionLogin             = "auth.login"
	ActionLogout            = "auth.logout"
	ActionPasswordChange    = "auth.password_change"
	ActionProfileUpdate     = "auth.profile_update"
	ActionSessionRevoke     = "auth.session_revoke"
	ActionDocumentUpload    = "document.upload"
	ActionDocumentDownload  = "document.download"
	ActionDocumentDelete    = "document.delete"
	ActionMarkdownSave      = "document.markdown_save"
	ActionOfficeSave        = "office.save"
	ActionVersionRestore    = "document.version_restore"
	ActionPermissionGrant   = "permission.grant"
	ActionPermissionDelete  = "permission.delete"
	ActionShareCreate       = "share.create"
	ActionShareDisable      = "share.disable"
	ActionShareAccess       = "share.access"
	ActionShareDownload     = "share.download"
	ActionShareMarkdownSave = "share.markdown_save"
)

type Log struct {
	ID          string
	ActorUserID *string
	Action      string
	TargetType  string
	TargetID    string
	IPAddr      *string
	UserAgent   *string
	Metadata    map[string]any
	CreatedAt   time.Time
}

type PublicLog struct {
	ID          string         `json:"id"`
	ActorUserID *string        `json:"actorUserId"`
	Action      string         `json:"action"`
	TargetType  string         `json:"targetType"`
	TargetID    string         `json:"targetId"`
	IPAddr      *string        `json:"ipAddr"`
	UserAgent   *string        `json:"userAgent"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type RecordInput struct {
	ActorUserID *string
	Action      string
	TargetType  string
	TargetID    string
	IPAddr      string
	UserAgent   string
	Metadata    map[string]any
}

type ListFilter struct {
	ActorUserID *string
	Action      string
	TargetType  string
	TargetID    string
	From        *time.Time
	To          *time.Time
	Limit       int
	Offset      int
}

func ToPublic(log Log) PublicLog {
	metadata := log.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	return PublicLog{
		ID:          log.ID,
		ActorUserID: log.ActorUserID,
		Action:      log.Action,
		TargetType:  log.TargetType,
		TargetID:    log.TargetID,
		IPAddr:      log.IPAddr,
		UserAgent:   log.UserAgent,
		Metadata:    metadata,
		CreatedAt:   log.CreatedAt,
	}
}
