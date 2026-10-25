//go:build devlogin

package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
)

// This file exists only in builds made with the devlogin tag (make
// dev-login). It replaces Google OAuth with a local form that signs in as any
// typed Google ID, so disposable accounts can exercise sign-up, export,
// deletion, and the deleted-account tombstone without a real Google account.
// Signing in as an arbitrary Google ID impersonates whoever owns it, so the
// default and App Engine builds leave this file out, and devLoginAllowed
// refuses any database that is not local.

// devLoginMaxIDLength bounds the typed Google ID. Real Google IDs are 21
// digits.
const devLoginMaxIDLength = 64

// devLoginAllowed reports why the dev login must not be registered in the
// current environment, or nil when it may be. It refuses App Engine, an
// explicit production environment, and a real Datastore: staging shares the
// production Datastore, so GOOGLE_CLOUD_PROJECT is accepted only with the
// emulator. Without GOOGLE_CLOUD_PROJECT the server uses the local SQLite
// file.
func devLoginAllowed() error {
	if env := os.Getenv("GAE_ENV"); env != "" {
		return fmt.Errorf("GAE_ENV is set to %q", env)
	}
	if os.Getenv("ENVIRONMENT") == "production" {
		return fmt.Errorf("ENVIRONMENT is production")
	}
	if os.Getenv("GOOGLE_CLOUD_PROJECT") != "" && os.Getenv("DATASTORE_EMULATOR_HOST") == "" {
		return fmt.Errorf("GOOGLE_CLOUD_PROJECT is set without DATASTORE_EMULATOR_HOST, so the database is a real Datastore")
	}
	return nil
}

// RegisterDevLogin adds the dev login routes to the /auth group and points
// Login at them instead of Google, when devLoginAllowed permits it. It
// reports whether the routes were registered.
func (ah *AuthHandler) RegisterDevLogin(authRoutes *gin.RouterGroup) bool {
	if err := devLoginAllowed(); err != nil {
		log.Printf("Dev login NOT registered: %v", err)
		return false
	}

	formPath := authRoutes.BasePath() + "/dev-login"
	ah.loginURL = func(state string) string {
		return formPath + "?state=" + url.QueryEscape(state)
	}
	authRoutes.GET("/dev-login", ah.DevLoginForm)
	authRoutes.GET("/dev-login/callback", ah.DevLoginCallback)

	log.Printf("WARNING: dev login is enabled; /auth/login signs in as any typed Google ID instead of using Google OAuth")
	return true
}

var devLoginTemplate = template.Must(template.New("dev-login").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Dev Login - GoRead2</title>
<style>
body { font-family: -apple-system, system-ui, sans-serif; max-width: 28rem; margin: 2rem auto; padding: 0 1rem; }
label { display: block; margin-top: 1rem; font-weight: 600; }
input { width: 100%; box-sizing: border-box; padding: 0.5rem; font-size: 1rem; }
button { margin-top: 1.5rem; padding: 0.6rem 1.2rem; font-size: 1rem; }
.note { color: #666; font-size: 0.9rem; }
.error { color: #b00020; }
</style>
</head>
<body>
<h1>Dev Login</h1>
<p class="note">Local development only. Signs in as the Google ID below without contacting Google. Reusing the ID of a deleted account tests the deleted-account tombstone.</p>
{{if .Error}}<p class="error" role="alert">{{.Error}}</p>{{end}}
<form method="get" action="{{.Action}}">
<input type="hidden" name="state" value="{{.State}}">
<label for="id">Google ID</label>
<input id="id" name="id" value="{{.ID}}" required maxlength="64" autocapitalize="off" autocorrect="off" spellcheck="false">
<label for="email">Email</label>
<input id="email" name="email" type="email" value="{{.Email}}" required autocapitalize="off" autocorrect="off" spellcheck="false">
<label for="name">Name</label>
<input id="name" name="name" value="{{.Name}}">
<button type="submit">Sign in</button>
</form>
</body>
</html>
`))

type devLoginForm struct {
	Action string
	State  string
	ID     string
	Email  string
	Name   string
	Error  string
}

// DevLoginForm serves the sign-in form Login redirects to in place of
// Google's consent page. Each visit pre-fills a fresh Google ID, so
// submitting unchanged creates a new account.
func (ah *AuthHandler) DevLoginForm(c *gin.Context) {
	suffix, err := randomHex(4)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication setup failed. Please try signing in again."})
		return
	}
	id := "dev-" + suffix
	ah.renderDevLoginForm(c, http.StatusOK, devLoginForm{
		State: c.Query("state"),
		ID:    id,
		Email: id + "@example.test",
		Name:  "Dev User " + suffix,
	})
}

// DevLoginCallback stands in for /auth/callback: it consumes the OAuth state
// Login issued, then signs in as the submitted identity through the same
// create-or-load, tombstone, and session code as the Google callback.
func (ah *AuthHandler) DevLoginCallback(c *gin.Context) {
	form := devLoginForm{
		State: c.Query("state"),
		ID:    strings.TrimSpace(c.Query("id")),
		Email: strings.TrimSpace(c.Query("email")),
		Name:  strings.TrimSpace(c.Query("name")),
	}
	// Validate before consuming the state, so a typo re-renders the form
	// and the same sign-in attempt can be resubmitted.
	if msg := validateDevLogin(form); msg != "" {
		form.Error = msg
		ah.renderDevLoginForm(c, http.StatusBadRequest, form)
		return
	}
	if form.Name == "" {
		form.Name = form.Email[:strings.Index(form.Email, "@")]
	}

	native, ok := ah.consumeOAuthState(c)
	if !ok {
		return
	}

	user, err := ah.authService.SignInGoogleUser(c.Request.Context(), auth.GoogleUserInfo{
		ID:    form.ID,
		Email: form.Email,
		Name:  form.Name,
	})
	if err != nil {
		log.Printf("Dev login error: %v", err)
		ah.callbackError(c, native, http.StatusInternalServerError, "Authentication failed. Please try signing in again.")
		return
	}
	log.Printf("Dev login: signed in user %d (native=%v)", user.ID, native)
	ah.completeSignIn(c, user, native)
}

// validateDevLogin returns a message describing the first invalid field, or
// "" when the form is valid.
func validateDevLogin(form devLoginForm) string {
	if form.ID == "" {
		return "A Google ID is required."
	}
	if len(form.ID) > devLoginMaxIDLength {
		return fmt.Sprintf("The Google ID must be at most %d characters.", devLoginMaxIDLength)
	}
	if addr, err := mail.ParseAddress(form.Email); err != nil || addr.Address != form.Email {
		return "A plain email address such as dev@example.test is required."
	}
	return ""
}

func (ah *AuthHandler) renderDevLoginForm(c *gin.Context, status int, form devLoginForm) {
	form.Action = strings.TrimSuffix(c.FullPath(), "/callback") + "/callback"
	// The native flow ends with a redirect to goread2://auth from the form's
	// submission, which the default form-action 'self' would block.
	c.Header("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; form-action 'self' goread2:")
	c.Header("Cache-Control", "no-store")
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := devLoginTemplate.Execute(c.Writer, form); err != nil {
		log.Printf("Dev login form failed to render: %v", err)
	}
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
