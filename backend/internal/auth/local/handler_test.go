package local

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"online-colab-document/backend/internal/auth/session"
	"online-colab-document/backend/internal/user"
)

func TestRegisterSuccess(t *testing.T) {
	handler, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/local/register", strings.NewReader(`{
		"email":"User@Example.com",
		"displayName":"User",
		"password":"password123"
	}`))
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var body user.PublicUser
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Email != "user@example.com" || body.AuthSource != "local" {
		t.Fatalf("unexpected user response: %#v", body)
	}
}

func TestRegisterDuplicateEmailFails(t *testing.T) {
	handler, _ := newTestHandler()
	body := `{"email":"user@example.com","displayName":"User","password":"password123"}`
	handler.Register(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body)))

	rec := httptest.NewRecorder()
	handler.Register(rec, httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body)))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Code)
	}
}

func TestLoginSuccess(t *testing.T) {
	handler, _ := newTestHandler()
	registerUser(t, handler, "user@example.com")
	rec := httptest.NewRecorder()

	handler.Login(rec, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{
		"email":"user@example.com",
		"password":"password123"
	}`)))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Result().Cookies()[0].Name != "docs_session" {
		t.Fatalf("expected session cookie, got %#v", rec.Result().Cookies())
	}
	if !rec.Result().Cookies()[0].HttpOnly {
		t.Fatalf("expected HttpOnly cookie")
	}
}

func TestLoginWrongPasswordFails(t *testing.T) {
	handler, _ := newTestHandler()
	registerUser(t, handler, "user@example.com")
	rec := httptest.NewRecorder()

	handler.Login(rec, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{
		"email":"user@example.com",
		"password":"wrong-password"
	}`)))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestDisabledUserLoginFails(t *testing.T) {
	handler, repo := newTestHandler()
	registerUser(t, handler, "user@example.com")
	repo.disable("user@example.com")
	rec := httptest.NewRecorder()

	handler.Login(rec, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{
		"email":"user@example.com",
		"password":"password123"
	}`)))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestMeUnauthenticatedReturns401(t *testing.T) {
	handler, _ := newTestHandler()
	rec := httptest.NewRecorder()

	handler.Me(rec, httptest.NewRequest(http.MethodGet, "/me", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestMeAuthenticatedReturnsUser(t *testing.T) {
	handler, _ := newTestHandler()
	registerUser(t, handler, "user@example.com")
	loginRec := httptest.NewRecorder()
	handler.Login(loginRec, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{
		"email":"user@example.com",
		"password":"password123"
	}`)))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(loginRec.Result().Cookies()[0])

	handler.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body user.PublicUser
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Email != "user@example.com" {
		t.Fatalf("expected user@example.com, got %q", body.Email)
	}
}

func TestMeAuthenticatedReturnsOIDCUser(t *testing.T) {
	handler, repo := newTestHandler()
	subject := "oidc-subject"
	oidcUser := user.User{
		ID:          "oidc-user-1",
		Email:       "oidc@example.com",
		DisplayName: "OIDC User",
		AuthSource:  "oidc",
		OIDCSubject: &subject,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repo.Create(context.Background(), oidcUser); err != nil {
		t.Fatalf("create oidc user: %v", err)
	}
	token := "oidc-session-token"
	if err := repo.CreateSession(context.Background(), session.Record{
		ID:        "session-1",
		UserID:    oidcUser.ID,
		TokenHash: session.HashToken(token),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("create oidc session: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "docs_session", Value: token})

	handler.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body user.PublicUser
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Email != oidcUser.Email || body.AuthSource != "oidc" {
		t.Fatalf("unexpected oidc user response: %#v", body)
	}
}

func newTestHandler() (Handler, *memoryRepo) {
	repo := newMemoryRepo()
	service := NewService(repo, repo, "", time.Hour)
	return NewHandler(service, slog.Default(), "docs_session", false), repo
}

func registerUser(t *testing.T, handler Handler, email string) {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.Register(rec, httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{
		"email":"`+email+`",
		"displayName":"User",
		"password":"password123"
	}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("register user: status %d, body %s", rec.Code, rec.Body.String())
	}
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
		return ErrEmailAlreadyUsed
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
		return user.User{}, ErrUserNotFound
	}
	return r.usersByID[id], nil
}

func (r *memoryRepo) FindByID(_ context.Context, id string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.usersByID[id]
	if !ok {
		return user.User{}, ErrUserNotFound
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
	return user.User{}, ErrUserNotFound
}

func (r *memoryRepo) SetOIDCSubject(_ context.Context, id string, subject string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.usersByID[id]
	if !ok {
		return user.User{}, ErrUserNotFound
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

func (r *memoryRepo) FindSessionByTokenHash(_ context.Context, tokenHash string) (session.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.sessionsByHash[tokenHash]
	if !ok {
		return session.Record{}, ErrSessionNotFound
	}
	return record, nil
}

func (r *memoryRepo) DeleteSessionByTokenHash(_ context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sessionsByHash[tokenHash]; !ok {
		return nil
	}
	delete(r.sessionsByHash, tokenHash)
	return nil
}

func (r *memoryRepo) disable(email string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.userIDByEmail[email]
	u := r.usersByID[id]
	u.Disabled = true
	r.usersByID[id] = u
}
