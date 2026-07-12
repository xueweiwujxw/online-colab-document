package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"online-colab-document/backend/internal/auth/local"
	"online-colab-document/backend/internal/auth/session"
	"online-colab-document/backend/internal/user"
)

var (
	ErrDisabled       = errors.New("oidc disabled")
	ErrInvalidState   = errors.New("invalid oidc state")
	ErrInvalidToken   = errors.New("invalid oidc token")
	ErrEmailConflict  = errors.New("email already registered")
	ErrProviderFailed = errors.New("oidc provider failed")
)

type Config struct {
	Enabled          bool
	IssuerURL        string
	ClientID         string
	ClientSecret     string
	RedirectURL      string
	Scopes           []string
	AutoMergeByEmail bool
}

type UserRepository interface {
	Create(ctx context.Context, user user.User) error
	FindByEmail(ctx context.Context, email string) (user.User, error)
	FindByID(ctx context.Context, id string) (user.User, error)
	FindByOIDCSubject(ctx context.Context, subject string) (user.User, error)
	SetOIDCSubject(ctx context.Context, id string, subject string) (user.User, error)
}

type SessionRepository interface {
	CreateSession(ctx context.Context, record session.Record) error
}

type Provider struct {
	OAuth    OAuthClient
	Verifier TokenVerifier
	UserInfo UserInfoFetcher
}

type OAuthClient interface {
	AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
}

type TokenVerifier interface {
	Verify(ctx context.Context, rawIDToken string) (IDToken, error)
}

type UserInfoFetcher interface {
	Fetch(ctx context.Context, tokenSource oauth2.TokenSource) (Profile, error)
}

type IDToken struct {
	Subject string
	Nonce   string
}

type Profile struct {
	Subject     string
	Email       string
	DisplayName string
}

type ProviderFactory func(ctx context.Context, cfg Config) (*Provider, error)

type Service struct {
	cfg             Config
	users           UserRepository
	sessions        SessionRepository
	providerFactory ProviderFactory
	now             func() time.Time
	newID           func() (string, error)
}

type AuthSession struct {
	User      user.User
	Token     string
	ExpiresAt time.Time
}

func NewService(cfg Config, users UserRepository, sessions SessionRepository, providerFactory ProviderFactory) *Service {
	return &Service{
		cfg:             cfg,
		users:           users,
		sessions:        sessions,
		providerFactory: providerFactory,
		now:             time.Now,
		newID:           newUUID,
	}
}

func (s *Service) Enabled() bool {
	return s.cfg.Enabled
}

func (s *Service) LoginURL(ctx context.Context, state string, nonce string) (string, error) {
	if !s.cfg.Enabled {
		return "", ErrDisabled
	}
	provider, err := s.provider(ctx)
	if err != nil {
		return "", err
	}
	return provider.OAuth.AuthCodeURL(state, Nonce(nonce)), nil
}

func (s *Service) Callback(ctx context.Context, code string, nonce string, sessionTTL time.Duration) (AuthSession, error) {
	if !s.cfg.Enabled {
		return AuthSession{}, ErrDisabled
	}
	if code == "" || nonce == "" {
		return AuthSession{}, ErrInvalidState
	}
	provider, err := s.provider(ctx)
	if err != nil {
		return AuthSession{}, err
	}
	token, err := provider.OAuth.Exchange(ctx, code)
	if err != nil {
		return AuthSession{}, fmt.Errorf("%w: exchange token", ErrProviderFailed)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return AuthSession{}, ErrInvalidToken
	}
	idToken, err := provider.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return AuthSession{}, ErrInvalidToken
	}
	if idToken.Nonce != nonce {
		return AuthSession{}, ErrInvalidToken
	}
	profile, err := provider.UserInfo.Fetch(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return AuthSession{}, fmt.Errorf("%w: fetch userinfo", ErrProviderFailed)
	}
	profile = normalizeProfile(profile, idToken.Subject)
	if profile.Subject == "" || profile.Email == "" {
		return AuthSession{}, ErrInvalidToken
	}
	u, err := s.upsertOIDCUser(ctx, profile)
	if err != nil {
		return AuthSession{}, err
	}
	return s.createSession(ctx, u, sessionTTL)
}

func (s *Service) provider(ctx context.Context) (*Provider, error) {
	if s.providerFactory == nil {
		return nil, ErrProviderFailed
	}
	provider, err := s.providerFactory(ctx, s.cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderFailed, err)
	}
	return provider, nil
}

func (s *Service) upsertOIDCUser(ctx context.Context, profile Profile) (user.User, error) {
	u, err := s.users.FindByOIDCSubject(ctx, profile.Subject)
	if err == nil {
		if u.Disabled {
			return user.User{}, local.ErrInvalidCredentials
		}
		return u, nil
	}
	if !errors.Is(err, local.ErrUserNotFound) {
		return user.User{}, err
	}

	existing, err := s.users.FindByEmail(ctx, profile.Email)
	if err == nil {
		if !s.cfg.AutoMergeByEmail {
			return user.User{}, ErrEmailConflict
		}
		if existing.Disabled {
			return user.User{}, local.ErrInvalidCredentials
		}
		return s.users.SetOIDCSubject(ctx, existing.ID, profile.Subject)
	}
	if err != nil && !errors.Is(err, local.ErrUserNotFound) {
		return user.User{}, err
	}

	id, err := s.newID()
	if err != nil {
		return user.User{}, err
	}
	now := s.now().UTC()
	oidcSubject := profile.Subject
	u = user.User{
		ID:           id,
		Email:        profile.Email,
		DisplayName:  profile.DisplayName,
		PasswordHash: nil,
		AuthSource:   "oidc",
		OIDCSubject:  &oidcSubject,
		IsAdmin:      false,
		Disabled:     false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, local.ErrEmailAlreadyUsed) {
			return user.User{}, ErrEmailConflict
		}
		return user.User{}, err
	}
	return u, nil
}

func (s *Service) createSession(ctx context.Context, u user.User, ttl time.Duration) (AuthSession, error) {
	token, err := session.NewToken()
	if err != nil {
		return AuthSession{}, err
	}
	sessionID, err := s.newID()
	if err != nil {
		return AuthSession{}, err
	}
	now := s.now().UTC()
	expiresAt := now.Add(ttl)
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

func normalizeProfile(profile Profile, subject string) Profile {
	profile.Subject = strings.TrimSpace(profile.Subject)
	if profile.Subject == "" {
		profile.Subject = subject
	}
	profile.Email = strings.ToLower(strings.TrimSpace(profile.Email))
	profile.DisplayName = strings.TrimSpace(profile.DisplayName)
	if profile.DisplayName == "" {
		profile.DisplayName = profile.Email
	}
	return profile
}

func NewStateToken() (string, error) {
	return randomToken(32)
}

func NewNonce() (string, error) {
	return randomToken(32)
}

func randomToken(size int) (string, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
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
