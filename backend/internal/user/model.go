package user

import "time"

type User struct {
	ID           string
	Email        string
	DisplayName  string
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
	AuthSource  string `json:"authSource"`
	IsAdmin     bool   `json:"isAdmin"`
}

func ToPublic(u User) PublicUser {
	return PublicUser{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AuthSource:  u.AuthSource,
		IsAdmin:     u.IsAdmin,
	}
}
