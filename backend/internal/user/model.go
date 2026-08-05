package user

import (
	"time"
)

type User struct {
	ID           string
	Email        string
	DisplayName  string
	AvatarKey    *string
	PasswordHash *string
	AuthSource   string
	OIDCSubject  *string
	IsAdmin      bool
	Disabled     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PublicUser struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	AuthSource  string `json:"authSource"`
	IsAdmin     bool   `json:"isAdmin"`
}

type AdminUser struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	AuthSource  string    `json:"authSource"`
	IsAdmin     bool      `json:"isAdmin"`
	Disabled    bool      `json:"disabled"`
	CreatedAt   time.Time `json:"createdAt"`
	AvatarURL   string    `json:"avatarUrl,omitempty"`
}

func ToPublic(u User) PublicUser {
	avatarURL := ""
	if u.AvatarKey != nil && *u.AvatarKey != "" {
		avatarURL = "/api/users/" + u.ID + "/avatar"
	}
	return PublicUser{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AuthSource:  u.AuthSource,
		IsAdmin:     u.IsAdmin,
		AvatarURL:   avatarURL,
	}
}

func ToAdmin(u User) AdminUser {
	public := ToPublic(u)
	return AdminUser{ID: public.ID, Email: public.Email, DisplayName: public.DisplayName, AuthSource: public.AuthSource, IsAdmin: public.IsAdmin, Disabled: u.Disabled, CreatedAt: u.CreatedAt, AvatarURL: public.AvatarURL}
}
