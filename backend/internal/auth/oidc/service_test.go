package oidc

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"online-colab-document/backend/internal/auth/local"
	"online-colab-document/backend/internal/auth/session"
	"online-colab-document/backend/internal/user"
)

func TestLoginDisabledReturnsUnavailable(t *testing.T) {
	service := NewService(Config{Enabled: false}, newMemoryRepo(), newMemoryRepo(), nil)

	_, err := service.LoginURL(context.Background(), "state", "nonce")

	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}

func TestLoginHandlerDisabledReturns404(t *testing.T) {
	handler := NewHandler(
		NewService(Config{Enabled: false}, newMemoryRepo(), newMemoryRepo(), nil),
		slog.Default(),
		"docs_session",
		false,
		time.Hour,
		"http://localhost:3000",
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)

	handler.Login(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestCallbackStateError(t *testing.T) {
	handler := NewHandler(
		NewService(Config{Enabled: true}, newMemoryRepo(), newMemoryRepo(), newFakeProviderFactory(fakeProvider{})),
		slog.Default(),
		"docs_session",
		false,
		time.Hour,
		"http://localhost:3000",
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback?state=actual&code=code", nil)
	req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "expected"})
	req.AddCookie(&http.Cookie{Name: nonceCookieName, Value: "nonce"})

	handler.Callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCallbackTokenVerificationFailure(t *testing.T) {
	repo := newMemoryRepo()
	service := NewService(Config{Enabled: true}, repo, repo, newFakeProviderFactory(fakeProvider{
		token:     (&oauth2.Token{AccessToken: "access"}).WithExtra(map[string]any{"id_token": "raw"}),
		verifyErr: errors.New("bad token"),
	}))

	_, err := service.Callback(context.Background(), "code", "nonce", time.Hour)

	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestCallbackCreatesNewOIDCUser(t *testing.T) {
	repo := newMemoryRepo()
	service := NewService(Config{Enabled: true}, repo, repo, newFakeProviderFactory(fakeProvider{
		token:   (&oauth2.Token{AccessToken: "access"}).WithExtra(map[string]any{"id_token": "raw"}),
		idToken: IDToken{Subject: "subject-1", Nonce: "nonce"},
		profile: Profile{Subject: "subject-1", Email: "User@Example.com", DisplayName: "OIDC User"},
	}))

	authSession, err := service.Callback(context.Background(), "code", "nonce", time.Hour)

	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if authSession.User.AuthSource != "oidc" {
		t.Fatalf("expected oidc auth source, got %q", authSession.User.AuthSource)
	}
	if authSession.User.Email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", authSession.User.Email)
	}
	if authSession.Token == "" {
		t.Fatalf("expected session token")
	}
}

func TestCallbackExistingOIDCUserLogin(t *testing.T) {
	repo := newMemoryRepo()
	subject := "subject-1"
	existing := user.User{
		ID:          "user-1",
		Email:       "user@example.com",
		DisplayName: "OIDC User",
		AuthSource:  "oidc",
		OIDCSubject: &subject,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repo.Create(context.Background(), existing); err != nil {
		t.Fatalf("create existing user: %v", err)
	}
	service := NewService(Config{Enabled: true}, repo, repo, newFakeProviderFactory(fakeProvider{
		token:   (&oauth2.Token{AccessToken: "access"}).WithExtra(map[string]any{"id_token": "raw"}),
		idToken: IDToken{Subject: subject, Nonce: "nonce"},
		profile: Profile{Subject: subject, Email: "user@example.com", DisplayName: "OIDC User"},
	}))

	authSession, err := service.Callback(context.Background(), "code", "nonce", time.Hour)

	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if authSession.User.ID != existing.ID {
		t.Fatalf("expected existing user %q, got %q", existing.ID, authSession.User.ID)
	}
	if repo.userCount() != 1 {
		t.Fatalf("expected one user, got %d", repo.userCount())
	}
}

func TestCallbackEmailConflictDoesNotAutoMerge(t *testing.T) {
	repo := newMemoryRepo()
	hash := "hash"
	if err := repo.Create(context.Background(), user.User{
		ID:           "local-1",
		Email:        "user@example.com",
		DisplayName:  "Local User",
		PasswordHash: &hash,
		AuthSource:   "local",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatalf("create local user: %v", err)
	}
	service := NewService(Config{Enabled: true, AutoMergeByEmail: false}, repo, repo, newFakeProviderFactory(fakeProvider{
		token:   (&oauth2.Token{AccessToken: "access"}).WithExtra(map[string]any{"id_token": "raw"}),
		idToken: IDToken{Subject: "subject-1", Nonce: "nonce"},
		profile: Profile{Subject: "subject-1", Email: "user@example.com", DisplayName: "OIDC User"},
	}))

	_, err := service.Callback(context.Background(), "code", "nonce", time.Hour)

	if !errors.Is(err, ErrEmailConflict) {
		t.Fatalf("expected ErrEmailConflict, got %v", err)
	}
}

type fakeProvider struct {
	token       *oauth2.Token
	exchangeErr error
	idToken     IDToken
	verifyErr   error
	profile     Profile
	userInfoErr error
}

func newFakeProviderFactory(fake fakeProvider) ProviderFactory {
	return func(context.Context, Config) (*Provider, error) {
		return &Provider{
			OAuth:    fakeOAuthClient{fake: fake},
			Verifier: fakeVerifier{fake: fake},
			UserInfo: fakeUserInfo{fake: fake},
		}, nil
	}
}

type fakeOAuthClient struct {
	fake fakeProvider
}

func (c fakeOAuthClient) AuthCodeURL(state string, _ ...oauth2.AuthCodeOption) string {
	return "https://idp.example.test/authorize?state=" + state
}

func (c fakeOAuthClient) Exchange(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	if c.fake.exchangeErr != nil {
		return nil, c.fake.exchangeErr
	}
	if c.fake.token != nil {
		return c.fake.token, nil
	}
	return (&oauth2.Token{AccessToken: "access"}).WithExtra(map[string]any{"id_token": "raw"}), nil
}

type fakeVerifier struct {
	fake fakeProvider
}

func (v fakeVerifier) Verify(context.Context, string) (IDToken, error) {
	if v.fake.verifyErr != nil {
		return IDToken{}, v.fake.verifyErr
	}
	return v.fake.idToken, nil
}

type fakeUserInfo struct {
	fake fakeProvider
}

func (u fakeUserInfo) Fetch(context.Context, oauth2.TokenSource) (Profile, error) {
	if u.fake.userInfoErr != nil {
		return Profile{}, u.fake.userInfoErr
	}
	return u.fake.profile, nil
}

type memoryRepo struct {
	mu             sync.Mutex
	usersByID      map[string]user.User
	userIDByEmail  map[string]string
	sessionsByHash map[string]session.Record
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		usersByID:      map[string]user.User{},
		userIDByEmail:  map[string]string{},
		sessionsByHash: map[string]session.Record{},
	}
}

func (r *memoryRepo) Create(_ context.Context, u user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.userIDByEmail[u.Email]; ok {
		return local.ErrEmailAlreadyUsed
	}
	r.usersByID[u.ID] = u
	r.userIDByEmail[u.Email] = u.ID
	return nil
}

func (r *memoryRepo) FindByEmail(_ context.Context, email string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.userIDByEmail[email]
	if !ok {
		return user.User{}, local.ErrUserNotFound
	}
	return r.usersByID[id], nil
}

func (r *memoryRepo) FindByID(_ context.Context, id string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.usersByID[id]
	if !ok {
		return user.User{}, local.ErrUserNotFound
	}
	return u, nil
}

func (r *memoryRepo) FindByOIDCSubject(_ context.Context, subject string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.usersByID {
		if u.OIDCSubject != nil && *u.OIDCSubject == subject {
			return u, nil
		}
	}
	return user.User{}, local.ErrUserNotFound
}

func (r *memoryRepo) SetOIDCSubject(_ context.Context, id string, subject string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.usersByID[id]
	if !ok {
		return user.User{}, local.ErrUserNotFound
	}
	u.OIDCSubject = &subject
	u.UpdatedAt = time.Now()
	r.usersByID[id] = u
	return u, nil
}

func (r *memoryRepo) CreateSession(_ context.Context, record session.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionsByHash[record.TokenHash] = record
	return nil
}

func (r *memoryRepo) userCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.usersByID)
}
