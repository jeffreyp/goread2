//go:build !devlogin

package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
)

// TestDefaultBuildHasNoDevLogin checks that a build without the devlogin tag,
// which includes the App Engine build, registers no dev login routes even
// in an environment the tagged build would accept, and still sends Login to
// Google.
func TestDefaultBuildHasNoDevLogin(t *testing.T) {
	for _, name := range []string{"GAE_ENV", "ENVIRONMENT", "GOOGLE_CLOUD_PROJECT", "DATASTORE_EMULATOR_HOST"} {
		t.Setenv(name, "")
	}
	gin.SetMode(gin.TestMode)
	db := &mockDBAuthHandler{}
	handler := NewAuthHandler(auth.NewAuthService(db), auth.NewSessionManager(db), auth.NewCSRFManager())
	r := gin.New()
	authRoutes := r.Group("/auth")
	authRoutes.GET("/login", handler.Login)

	if handler.RegisterDevLogin(authRoutes) {
		t.Fatal("RegisterDevLogin reported success in a build without the devlogin tag")
	}
	for _, route := range r.Routes() {
		if route.Path != "/auth/login" {
			t.Errorf("unexpected route %s %s", route.Method, route.Path)
		}
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/auth/login?client=ios", nil))
	location, err := url.Parse(w.Header().Get("Location"))
	if w.Code != http.StatusFound || err != nil || location.Host != "accounts.google.com" {
		t.Errorf("expected Login to redirect to Google, got %d %q", w.Code, w.Header().Get("Location"))
	}
}
