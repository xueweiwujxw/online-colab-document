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
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthenticated    = errors.New("unauthenticated")
)

type SessionRepository interface {
	CreateSession(ctx context.Context, record session.Record) error
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (session.Record, error)
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

type AuthSession struct {
	User      user.User
	Token     string
	ExpiresAt time.Time
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
