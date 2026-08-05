package local

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"online-colab-document/backend/internal/audit"
	"online-colab-document/backend/internal/auth/session"
	"online-colab-document/backend/internal/middleware"
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

func TestLoginWritesAudit(t *testing.T) {
	handler, _ := newTestHandler()
	recorder := &fakeAuditRecorder{}
	handler = handler.WithAudit(recorder)
	registerUser(t, handler, "user@example.com")
	rec := httptest.NewRecorder()

	handler.Login(rec, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{
		"email":"user@example.com",
		"password":"password123"
	}`)))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(recorder.inputs) != 1 {
		t.Fatalf("expected one audit record, got %d", len(recorder.inputs))
	}
	if recorder.inputs[0].Action != audit.ActionLogin || recorder.inputs[0].TargetType != "user" {
		t.Fatalf("unexpected audit input: %#v", recorder.inputs[0])
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

func TestChangePasswordSuccess(t *testing.T) {
	handler, _ := newTestHandler()
	registerUser(t, handler, "user@example.com")
	loginRec := loginUser(t, handler, "user@example.com", "password123")

	rec := performPasswordChange(handler, loginRec.Result().Cookies()[0], `{
		"currentPassword":"password123",
		"newPassword":"new-password-123"
	}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	oldPasswordRec := httptest.NewRecorder()
	handler.Login(oldPasswordRec, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{
		"email":"user@example.com",
		"password":"password123"
	}`)))
	if oldPasswordRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected old password login to fail, got %d", oldPasswordRec.Code)
	}
	newPasswordRec := httptest.NewRecorder()
	handler.Login(newPasswordRec, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{
		"email":"user@example.com",
		"password":"new-password-123"
	}`)))
	if newPasswordRec.Code != http.StatusOK {
		t.Fatalf("expected new password login to succeed, got %d: %s", newPasswordRec.Code, newPasswordRec.Body.String())
	}
}

func TestChangePasswordWrongCurrentPasswordFails(t *testing.T) {
	handler, _ := newTestHandler()
	registerUser(t, handler, "user@example.com")
	loginRec := loginUser(t, handler, "user@example.com", "password123")

	rec := performPasswordChange(handler, loginRec.Result().Cookies()[0], `{
		"currentPassword":"wrong-password",
		"newPassword":"new-password-123"
	}`)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestChangePasswordOIDCUserFails(t *testing.T) {
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

	rec := performPasswordChange(handler, &http.Cookie{Name: "docs_session", Value: token}, `{
		"currentPassword":"password123",
		"newPassword":"new-password-123"
	}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminResetPasswordOnlySupportsLocalAccounts(t *testing.T) {
	_, repo := newTestHandler()
	now := time.Now().UTC()
	localHash, err := HashPassword("password123", "")
	if err != nil {
		t.Fatal(err)
	}
	repo.usersByID["local-1"] = user.User{ID: "local-1", Email: "local@example.test", AuthSource: "local", PasswordHash: &localHash, CreatedAt: now, UpdatedAt: now}
	service := NewService(repo, repo, "", time.Hour)
	admin := user.User{ID: "admin-1", IsAdmin: true}
	if err := service.AdminResetPassword(context.Background(), AdminResetPasswordInput{Actor: admin, TargetUserID: "local-1", NewPassword: "newpassword123"}); err != nil {
		t.Fatalf("reset local password: %v", err)
	}
	updated, _ := repo.FindByID(context.Background(), "local-1")
	if updated.PasswordHash == nil || !VerifyPassword("newpassword123", "", *updated.PasswordHash) {
		t.Fatal("expected reset password hash")
	}
	oidc := user.User{ID: "oidc-1", AuthSource: "oidc", CreatedAt: now, UpdatedAt: now}
	repo.usersByID[oidc.ID] = oidc
	if err := service.AdminResetPassword(context.Background(), AdminResetPasswordInput{Actor: admin, TargetUserID: oidc.ID, NewPassword: "newpassword123"}); !errors.Is(err, ErrPasswordUnsupported) {
		t.Fatalf("expected OIDC reset rejection, got %v", err)
	}
	if err := service.AdminResetPassword(context.Background(), AdminResetPasswordInput{Actor: user.User{}, TargetUserID: "local-1", NewPassword: "newpassword123"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected admin rejection, got %v", err)
	}
}

func TestUpdateProfileSuccess(t *testing.T) {
	handler, _ := newTestHandler()
	registerUser(t, handler, "user@example.com")
	loginRec := loginUser(t, handler, "user@example.com", "password123")

	rec := performProfileUpdate(handler, loginRec.Result().Cookies()[0], `{
		"displayName":"Updated User"
	}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body user.PublicUser
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.DisplayName != "Updated User" {
		t.Fatalf("expected updated display name, got %q", body.DisplayName)
	}

	meRec := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	meReq.AddCookie(loginRec.Result().Cookies()[0])
	handler.Me(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("expected me status 200, got %d: %s", meRec.Code, meRec.Body.String())
	}
	var meBody user.PublicUser
	if err := json.NewDecoder(meRec.Body).Decode(&meBody); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if meBody.DisplayName != "Updated User" {
		t.Fatalf("expected me display name to update, got %q", meBody.DisplayName)
	}
}

func TestUpdateProfileBlankDisplayNameFails(t *testing.T) {
	handler, _ := newTestHandler()
	registerUser(t, handler, "user@example.com")
	loginRec := loginUser(t, handler, "user@example.com", "password123")

	rec := performProfileUpdate(handler, loginRec.Result().Cookies()[0], `{
		"displayName":"   "
	}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListSessionsMarksCurrentSession(t *testing.T) {
	handler, _ := newTestHandler()
	registerUser(t, handler, "user@example.com")
	firstLogin := loginUser(t, handler, "user@example.com", "password123")
	loginUser(t, handler, "user@example.com", "password123")

	rec := performSessionList(handler, firstLogin.Result().Cookies()[0])

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items []PublicSession `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(body.Items))
	}
	currentCount := 0
	for _, item := range body.Items {
		if item.Current {
			currentCount++
		}
	}
	if currentCount != 1 {
		t.Fatalf("expected exactly one current session, got %d in %#v", currentCount, body.Items)
	}
}

func TestRevokeOtherSessionSuccess(t *testing.T) {
	handler, _ := newTestHandler()
	registerUser(t, handler, "user@example.com")
	currentLogin := loginUser(t, handler, "user@example.com", "password123")
	otherLogin := loginUser(t, handler, "user@example.com", "password123")
	otherSessionID := sessionIDForCookie(t, handler, otherLogin.Result().Cookies()[0])

	rec := performSessionRevoke(handler, currentLogin.Result().Cookies()[0], otherSessionID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	meRec := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	meReq.AddCookie(otherLogin.Result().Cookies()[0])
	handler.Me(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked session to fail, got %d", meRec.Code)
	}
}

func TestRevokeCurrentSessionFails(t *testing.T) {
	handler, _ := newTestHandler()
	registerUser(t, handler, "user@example.com")
	currentLogin := loginUser(t, handler, "user@example.com", "password123")
	currentSessionID := sessionIDForCookie(t, handler, currentLogin.Result().Cookies()[0])

	rec := performSessionRevoke(handler, currentLogin.Result().Cookies()[0], currentSessionID)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
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

func loginUser(t *testing.T, handler Handler, email string, password string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.Login(rec, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{
		"email":"`+email+`",
		"password":"`+password+`"
	}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("login user: status %d, body %s", rec.Code, rec.Body.String())
	}
	return rec
}

func performPasswordChange(handler Handler, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/auth/password", strings.NewReader(body))
	req.AddCookie(cookie)
	middleware.RequireAuth(
		handler.service,
		"docs_session",
		http.HandlerFunc(handler.ChangePassword),
	).ServeHTTP(rec, req)
	return rec
}

func performProfileUpdate(handler Handler, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/auth/profile", strings.NewReader(body))
	req.AddCookie(cookie)
	middleware.RequireAuth(
		handler.service,
		"docs_session",
		http.HandlerFunc(handler.UpdateProfile),
	).ServeHTTP(rec, req)
	return rec
}

func performSessionList(handler Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/sessions", nil)
	req.AddCookie(cookie)
	middleware.RequireAuth(
		handler.service,
		"docs_session",
		http.HandlerFunc(handler.ListSessions),
	).ServeHTTP(rec, req)
	return rec
}

func performSessionRevoke(handler Handler, cookie *http.Cookie, sessionID string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/"+sessionID, nil)
	req.SetPathValue("id", sessionID)
	req.AddCookie(cookie)
	middleware.RequireAuth(
		handler.service,
		"docs_session",
		http.HandlerFunc(handler.RevokeSession),
	).ServeHTTP(rec, req)
	return rec
}

func sessionIDForCookie(t *testing.T, handler Handler, cookie *http.Cookie) string {
	t.Helper()
	record, err := handler.service.sessions.FindSessionByTokenHash(context.Background(), session.HashToken(cookie.Value))
	if err != nil {
		t.Fatalf("find session for cookie: %v", err)
	}
	return record.ID
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

func (r *memoryRepo) UpdatePasswordHash(_ context.Context, id string, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.usersByID[id]
	if !ok {
		return ErrUserNotFound
	}
	u.PasswordHash = &passwordHash
	u.UpdatedAt = time.Now()
	r.usersByID[id] = u
	return nil
}

func (r *memoryRepo) UpdateDisplayName(_ context.Context, id string, displayName string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.usersByID[id]
	if !ok {
		return user.User{}, ErrUserNotFound
	}
	u.DisplayName = displayName
	u.UpdatedAt = time.Now()
	r.usersByID[id] = u
	return u, nil
}

func (r *memoryRepo) UpdateAvatarKey(_ context.Context, id string, avatarKey *string) (user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.usersByID[id]
	if !ok {
		return user.User{}, ErrUserNotFound
	}
	u.AvatarKey = avatarKey
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

func (r *memoryRepo) FindSessionByID(_ context.Context, id string) (session.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, record := range r.sessionsByHash {
		if record.ID == id {
			return record, nil
		}
	}
	return session.Record{}, ErrSessionNotFound
}

func (r *memoryRepo) ListSessionsByUserID(_ context.Context, userID string) ([]session.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	records := []session.Record{}
	now := time.Now()
	for _, record := range r.sessionsByHash {
		if record.UserID == userID && record.ExpiresAt.After(now) {
			records = append(records, record)
		}
	}
	return records, nil
}

func (r *memoryRepo) DeleteSessionByID(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for tokenHash, record := range r.sessionsByHash {
		if record.ID == id {
			delete(r.sessionsByHash, tokenHash)
			return nil
		}
	}
	return nil
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

type fakeAuditRecorder struct {
	inputs []audit.RecordInput
}

func (r *fakeAuditRecorder) Record(_ context.Context, input audit.RecordInput) error {
	r.inputs = append(r.inputs, input)
	return nil
}
