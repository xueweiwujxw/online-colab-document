package local

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"online-colab-document/backend/internal/auth/session"
	"online-colab-document/backend/internal/user"
)

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrUnauthenticated     = errors.New("unauthenticated")
	ErrPasswordUnsupported = errors.New("password change unsupported for this account")
	ErrSessionForbidden    = errors.New("session forbidden")
	ErrCurrentSession      = errors.New("cannot revoke current session")
)

type SessionRepository interface {
	CreateSession(ctx context.Context, record session.Record) error
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (session.Record, error)
	FindSessionByID(ctx context.Context, id string) (session.Record, error)
	ListSessionsByUserID(ctx context.Context, userID string) ([]session.Record, error)
	DeleteSessionByID(ctx context.Context, id string) error
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
}

type Service struct {
	users          UserRepository
	sessions       SessionRepository
	pepper         string
	sessionTTL     time.Duration
	now            func() time.Time
	newID          func() (string, error)
	hashPassword   func(string, string) (string, error)
	verifyPassword func(string, string, string) bool
}

func NewService(users UserRepository, sessions SessionRepository, pepper string, sessionTTL time.Duration) *Service {
	return &Service{
		users:          users,
		sessions:       sessions,
		pepper:         pepper,
		sessionTTL:     sessionTTL,
		now:            time.Now,
		newID:          newUUID,
		hashPassword:   HashPassword,
		verifyPassword: VerifyPassword,
	}
}

type RegisterInput struct {
	Email       string
	DisplayName string
	Password    string
}

type LoginInput struct {
	Email    string
	Password string
}

type ChangePasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

type UpdateProfileInput struct {
	UserID      string
	DisplayName string
}

type UpdateAvatarInput struct {
	UserID    string
	AvatarKey string
}

type AdminResetPasswordInput struct {
	Actor        user.User
	TargetUserID string
	NewPassword  string
}

type AdminUpdateUserInput struct {
	Actor        user.User
	TargetUserID string
	DisplayName  *string
	Email        *string
	IsAdmin      *bool
	Disabled     *bool
}

type AuthSession struct {
	User      user.User
	Token     string
	ExpiresAt time.Time
}

type PublicSession struct {
	ID        string    `json:"id"`
	Current   bool      `json:"current"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (user.User, error) {
	email := normalizeEmail(input.Email)
	displayName := strings.TrimSpace(input.DisplayName)
	if !validEmail(email) || displayName == "" || len(input.Password) < 8 {
		return user.User{}, ErrInvalidInput
	}
	passwordHash, err := s.hashPassword(input.Password, s.pepper)
	if err != nil {
		return user.User{}, err
	}
	id, err := s.newID()
	if err != nil {
		return user.User{}, err
	}
	now := s.now().UTC()
	u := user.User{
		ID:           id,
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: &passwordHash,
		AuthSource:   "local",
		IsAdmin:      false,
		Disabled:     false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return user.User{}, err
	}
	return u, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (AuthSession, error) {
	email := normalizeEmail(input.Email)
	if !validEmail(email) || input.Password == "" {
		return AuthSession{}, ErrInvalidCredentials
	}
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return AuthSession{}, ErrInvalidCredentials
		}
		return AuthSession{}, err
	}
	if u.Disabled || u.AuthSource != "local" || u.PasswordHash == nil {
		return AuthSession{}, ErrInvalidCredentials
	}
	if !s.verifyPassword(input.Password, s.pepper, *u.PasswordHash) {
		return AuthSession{}, ErrInvalidCredentials
	}
	token, err := session.NewToken()
	if err != nil {
		return AuthSession{}, err
	}
	sessionID, err := s.newID()
	if err != nil {
		return AuthSession{}, err
	}
	now := s.now().UTC()
	expiresAt := now.Add(s.sessionTTL)
	record := session.Record{
		ID:        sessionID,
		UserID:    u.ID,
		TokenHash: session.HashToken(token),
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	if err := s.sessions.CreateSession(ctx, record); err != nil {
		return AuthSession{}, err
	}
	return AuthSession{User: u, Token: token, ExpiresAt: expiresAt}, nil
}

func (s *Service) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	if input.UserID == "" || input.CurrentPassword == "" || len(input.NewPassword) < 8 {
		return ErrInvalidInput
	}
	u, err := s.users.FindByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrUnauthenticated
		}
		return err
	}
	if u.Disabled {
		return ErrUnauthenticated
	}
	if u.AuthSource != "local" || u.PasswordHash == nil {
		return ErrPasswordUnsupported
	}
	if !s.verifyPassword(input.CurrentPassword, s.pepper, *u.PasswordHash) {
		return ErrInvalidCredentials
	}
	nextHash, err := s.hashPassword(input.NewPassword, s.pepper)
	if err != nil {
		return err
	}
	return s.users.UpdatePasswordHash(ctx, u.ID, nextHash)
}

func (s *Service) UpdateProfile(ctx context.Context, input UpdateProfileInput) (user.User, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	if input.UserID == "" || displayName == "" || len(displayName) > 120 {
		return user.User{}, ErrInvalidInput
	}
	u, err := s.users.FindByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return user.User{}, ErrUnauthenticated
		}
		return user.User{}, err
	}
	if u.Disabled {
		return user.User{}, ErrUnauthenticated
	}
	return s.users.UpdateDisplayName(ctx, u.ID, displayName)
}

func (s *Service) UpdateAvatar(ctx context.Context, input UpdateAvatarInput) (user.User, error) {
	if input.UserID == "" || input.AvatarKey == "" {
		return user.User{}, ErrInvalidInput
	}
	u, err := s.users.FindByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return user.User{}, ErrUnauthenticated
		}
		return user.User{}, err
	}
	if u.Disabled {
		return user.User{}, ErrUnauthenticated
	}
	return s.users.UpdateAvatarKey(ctx, u.ID, &input.AvatarKey)
}

func (s *Service) AdminResetPassword(ctx context.Context, input AdminResetPasswordInput) error {
	if !input.Actor.IsAdmin || input.TargetUserID == "" || len(input.NewPassword) < 8 {
		return ErrInvalidInput
	}
	target, err := s.users.FindByID(ctx, input.TargetUserID)
	if err != nil {
		return err
	}
	if target.AuthSource != "local" || target.PasswordHash == nil {
		return ErrPasswordUnsupported
	}
	hash, err := s.hashPassword(input.NewPassword, s.pepper)
	if err != nil {
		return err
	}
	return s.users.UpdatePasswordHash(ctx, target.ID, hash)
}

func (s *Service) AdminUpdateUser(ctx context.Context, input AdminUpdateUserInput) (user.User, error) {
	if !input.Actor.IsAdmin || input.TargetUserID == "" {
		return user.User{}, ErrInvalidInput
	}
	if input.TargetUserID == input.Actor.ID && ((input.IsAdmin != nil && !*input.IsAdmin) || (input.Disabled != nil && *input.Disabled)) {
		return user.User{}, ErrInvalidInput
	}
	target, err := s.users.FindByID(ctx, input.TargetUserID)
	if err != nil {
		return user.User{}, err
	}
	if input.DisplayName != nil || input.Email != nil {
		if target.AuthSource != "local" {
			return user.User{}, ErrPasswordUnsupported
		}
	}
	if input.DisplayName != nil {
		name := strings.TrimSpace(*input.DisplayName)
		if name == "" || len(name) > 120 {
			return user.User{}, ErrInvalidInput
		}
		input.DisplayName = &name
	}
	if input.Email != nil {
		email := normalizeEmail(*input.Email)
		if !validEmail(email) {
			return user.User{}, ErrInvalidInput
		}
		input.Email = &email
	}
	return s.users.AdminUpdateUser(ctx, target.ID, input.DisplayName, input.Email, input.IsAdmin, input.Disabled)
}

func (s *Service) ListSessions(ctx context.Context, userID string, currentToken string) ([]PublicSession, error) {
	if userID == "" || currentToken == "" {
		return nil, ErrUnauthenticated
	}
	records, err := s.sessions.ListSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	currentHash := session.HashToken(currentToken)
	items := make([]PublicSession, 0, len(records))
	for _, record := range records {
		items = append(items, PublicSession{
			ID:        record.ID,
			Current:   record.TokenHash == currentHash,
			ExpiresAt: record.ExpiresAt,
			CreatedAt: record.CreatedAt,
		})
	}
	return items, nil
}

func (s *Service) RevokeSession(ctx context.Context, userID string, sessionID string, currentToken string) error {
	if userID == "" || sessionID == "" || currentToken == "" {
		return ErrInvalidInput
	}
	record, err := s.sessions.FindSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return ErrSessionForbidden
		}
		return err
	}
	if record.UserID != userID {
		return ErrSessionForbidden
	}
	if record.TokenHash == session.HashToken(currentToken) {
		return ErrCurrentSession
	}
	return s.sessions.DeleteSessionByID(ctx, sessionID)
}

func (s *Service) CurrentUser(ctx context.Context, token string) (user.User, error) {
	if token == "" {
		return user.User{}, ErrUnauthenticated
	}
	record, err := s.sessions.FindSessionByTokenHash(ctx, session.HashToken(token))
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return user.User{}, ErrUnauthenticated
		}
		return user.User{}, err
	}
	if !record.ExpiresAt.After(s.now()) {
		_ = s.sessions.DeleteSessionByTokenHash(ctx, record.TokenHash)
		return user.User{}, ErrUnauthenticated
	}
	u, err := s.users.FindByID(ctx, record.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return user.User{}, ErrUnauthenticated
		}
		return user.User{}, err
	}
	if u.Disabled {
		return user.User{}, ErrUnauthenticated
	}
	return u, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.DeleteSessionByTokenHash(ctx, session.HashToken(token))
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func validEmail(email string) bool {
	return emailPattern.MatchString(email)
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
