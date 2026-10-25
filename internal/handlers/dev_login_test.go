//go:build devlogin

package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/secrets"
)

// devLoginDB keeps users in memory and reports a tombstone for one ID, which
// is enough for the create-or-load and tombstone logic the dev login shares
// with the Google callback.
type devLoginDB struct {
	mockDBAuthHandler
	users       map[string]*database.User
	tombstoneID string
}

func newDevLoginDB() *devLoginDB {
	return &devLoginDB{users: map[string]*database.User{}}
}

func (db *devLoginDB) GetUserByGoogleID(googleID string) (*database.User, error) {
	if user, ok := db.users[googleID]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (db *devLoginDB) CreateUser(user *database.User) error {
	user.ID = len(db.users) + 1
	db.users[user.GoogleID] = user
	return nil
}

func (db *devLoginDB) HasAccountTombstone(id string) (bool, error) {
	return id != "" && id == db.tombstoneID, nil
}

// clearDevLoginGuardEnv unsets every variable devLoginAllowed reads, so each
// test starts from a plain local environment.
func clearDevLoginGuardEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"GAE_ENV", "ENVIRONMENT", "GOOGLE_CLOUD_PROJECT", "DATASTORE_EMULATOR_HOST"} {
		t.Setenv(name, "")
	}
}

func TestDevLoginAllowed(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		allowed bool
	}{
		{"local SQLite", nil, true},
		{"Datastore emulator", map[string]string{"GOOGLE_CLOUD_PROJECT": "goread2-local-test", "DATASTORE_EMULATOR_HOST": "localhost:8089"}, true},
		{"real Datastore", map[string]string{"GOOGLE_CLOUD_PROJECT": "goread-467200"}, false},
		{"App Engine", map[string]string{"GAE_ENV": "standard"}, false},
		{"App Engine with emulator", map[string]string{"GAE_ENV": "standard", "GOOGLE_CLOUD_PROJECT": "p", "DATASTORE_EMULATOR_HOST": "localhost:8089"}, false},
		{"production environment", map[string]string{"ENVIRONMENT": "production"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearDevLoginGuardEnv(t)
			for name, value := range tt.env {
				t.Setenv(name, value)
			}
			err := devLoginAllowed()
			if (err == nil) != tt.allowed {
				t.Errorf("devLoginAllowed() = %v, want allowed=%v", err, tt.allowed)
			}
		})
	}
}

// newDevLoginRouter builds the /auth routes main registers, with the dev
// login added.
func newDevLoginRouter(t *testing.T, db database.Database) (*gin.Engine, *AuthHandler, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(auth.NewAuthService(db), auth.NewSessionManager(db), auth.NewCSRFManager())
	r := gin.New()
	authRoutes := r.Group("/auth")
	authRoutes.GET("/login", handler.Login)
	authRoutes.GET("/callback", handler.Callback)
	registered := handler.RegisterDevLogin(authRoutes)
	return r, handler, registered
}

func hasRoute(r *gin.Engine, path string) bool {
	for _, route := range r.Routes() {
		if route.Path == path {
			return true
		}
	}
	return false
}

func TestRegisterDevLoginRefusedOnAppEngine(t *testing.T) {
	clearDevLoginGuardEnv(t)
	gin.SetMode(gin.TestMode)
	db := newDevLoginDB()
	// Built before GAE_ENV is set: the CSRF manager exits the process when
	// it finds App Engine without a CSRF secret.
	handler := NewAuthHandler(auth.NewAuthService(db), auth.NewSessionManager(db), auth.NewCSRFManager())
	r := gin.New()
	t.Setenv("GAE_ENV", "standard")

	registered := handler.RegisterDevLogin(r.Group("/auth"))
	if registered || handler.loginURL != nil {
		t.Fatal("expected the dev login to be refused on App Engine")
	}
	if hasRoute(r, "/auth/dev-login") || hasRoute(r, "/auth/dev-login/callback") {
		t.Error("expected no dev login routes on App Engine")
	}
}

// startDevLogin runs /auth/login and returns the state and its cookie.
func startDevLogin(t *testing.T, r *gin.Engine, query string) (string, *http.Cookie) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/auth/login"+query, nil))
	var stateCookie *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if strings.HasPrefix(cookie.Name, "oauth_state") {
			stateCookie = cookie
		}
	}
	if stateCookie == nil {
		t.Fatalf("login set no state cookie: %d %s", w.Code, w.Body.String())
	}
	state, err := url.QueryUnescape(stateCookie.Value)
	if err != nil {
		t.Fatalf("bad state cookie: %v", err)
	}
	return state, stateCookie
}

func devLoginCallback(r *gin.Engine, stateCookie *http.Cookie, values url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/auth/dev-login/callback?"+values.Encode(), nil)
	req.AddCookie(stateCookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDevLoginWebFlow(t *testing.T) {
	clearDevLoginGuardEnv(t)
	db := newDevLoginDB()
	r, _, registered := newDevLoginRouter(t, db)
	if !registered {
		t.Fatal("expected the dev login to register in a local environment")
	}

	state, stateCookie := startDevLogin(t, r, "")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/auth/dev-login?state="+url.QueryEscape(state), nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `action="/auth/dev-login/callback"`) {
		t.Fatalf("expected the form, got %d: %s", w.Code, w.Body.String())
	}
	if csp := w.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "form-action 'self' goread2:") {
		t.Errorf("form CSP must allow the goread2: handoff, got %q", csp)
	}

	w = devLoginCallback(r, stateCookie, url.Values{
		"state": {state}, "id": {"dev-web"}, "email": {"web@example.test"},
	})
	if w.Code != http.StatusTemporaryRedirect || w.Header().Get("Location") != "/" {
		t.Fatalf("expected a redirect to /, got %d %q", w.Code, w.Header().Get("Location"))
	}
	var sessionCookie *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if strings.HasPrefix(cookie.Name, "session_id") {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Errorf("expected a session cookie, got %q", w.Header().Values("Set-Cookie"))
	}
	user := db.users["dev-web"]
	if user == nil || user.Email != "web@example.test" || user.Name != "web" {
		t.Fatalf("expected user dev-web named after the email, got %+v", user)
	}

	// The state is single-use, exactly as on the Google callback.
	w = devLoginCallback(r, stateCookie, url.Values{
		"state": {state}, "id": {"dev-web"}, "email": {"web@example.test"},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected a replayed state to be rejected, got %d", w.Code)
	}
}

func TestDevLoginNativeFlow(t *testing.T) {
	clearDevLoginGuardEnv(t)
	r, handler, _ := newDevLoginRouter(t, newDevLoginDB())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/auth/login?client=ios", nil))
	location, err := url.Parse(w.Header().Get("Location"))
	if w.Code != http.StatusFound || err != nil || location.Path != "/auth/dev-login" {
		t.Fatalf("expected a redirect to the dev login form, got %d %q", w.Code, w.Header().Get("Location"))
	}
	state := location.Query().Get("state")
	var stateCookie *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if strings.HasPrefix(cookie.Name, "oauth_state") {
			stateCookie = cookie
		}
	}

	w = devLoginCallback(r, stateCookie, url.Values{
		"state": {state}, "id": {"dev-ios"}, "email": {"ios@example.test"}, "name": {"iOS Tester"},
	})
	callback, err := url.Parse(w.Header().Get("Location"))
	if w.Code != http.StatusFound || err != nil || callback.Scheme != "goread2" || callback.Query().Get("code") == "" {
		t.Fatalf("expected the goread2://auth code handoff, got %d %q", w.Code, w.Header().Get("Location"))
	}
	if _, _, ok := handler.sessionManager.ExchangeAuthCode(callback.Query().Get("code")); !ok {
		t.Error("expected the handoff code to exchange for a session")
	}
}

func TestDevLoginTombstonedIDStartsWithTrialEnded(t *testing.T) {
	clearDevLoginGuardEnv(t)
	t.Setenv("ACCOUNT_TOMBSTONE_KEY", "test-tombstone-key")
	secrets.ResetCacheForTesting()
	t.Cleanup(secrets.ResetCacheForTesting)

	db := newDevLoginDB()
	tombstoneID, err := secrets.AccountTombstoneID(context.Background(), "dev-deleted")
	if err != nil || tombstoneID == "" {
		t.Fatalf("failed to compute the tombstone ID: %q, %v", tombstoneID, err)
	}
	db.tombstoneID = tombstoneID
	r, _, _ := newDevLoginRouter(t, db)

	for _, id := range []string{"dev-deleted", "dev-fresh"} {
		state, stateCookie := startDevLogin(t, r, "")
		devLoginCallback(r, stateCookie, url.Values{"state": {state}, "id": {id}, "email": {id + "@example.test"}})
	}

	if deleted := db.users["dev-deleted"]; deleted == nil || !deleted.TrialEndsAt.Equal(deleted.CreatedAt) {
		t.Errorf("expected the tombstoned ID to get an ended trial, got %+v", deleted)
	}
	if fresh := db.users["dev-fresh"]; fresh == nil || !fresh.TrialEndsAt.IsZero() {
		t.Errorf("expected a new ID to keep the default trial, got %+v", fresh)
	}
}

func TestDevLoginInvalidInputKeepsState(t *testing.T) {
	clearDevLoginGuardEnv(t)
	db := newDevLoginDB()
	r, _, _ := newDevLoginRouter(t, db)
	state, stateCookie := startDevLogin(t, r, "")

	for name, values := range map[string]url.Values{
		"missing id":    {"state": {state}, "email": {"a@example.test"}},
		"long id":       {"state": {state}, "id": {strings.Repeat("x", devLoginMaxIDLength+1)}, "email": {"a@example.test"}},
		"bad email":     {"state": {state}, "id": {"dev-a"}, "email": {"not-an-email"}},
		"display email": {"state": {state}, "id": {"dev-a"}, "email": {"A <a@example.test>"}},
	} {
		t.Run(name, func(t *testing.T) {
			w := devLoginCallback(r, stateCookie, values)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `role="alert"`) {
				t.Errorf("expected the form again with an error, got %d: %s", w.Code, w.Body.String())
			}
		})
	}

	// None of the rejected attempts consumed the state.
	w := devLoginCallback(r, stateCookie, url.Values{"state": {state}, "id": {"dev-a"}, "email": {"a@example.test"}})
	if w.Code != http.StatusTemporaryRedirect || db.users["dev-a"] == nil {
		t.Errorf("expected the corrected attempt to sign in, got %d", w.Code)
	}
}
