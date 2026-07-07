package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/services"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stripe/stripe-go/v78"
)

// setupTestDB creates an in-memory SQLite database with all tables, mirroring the
// pattern used in internal/services tests.
func setupTestDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	dbWrapper := &database.DB{DB: db}
	if err := dbWrapper.CreateTables(); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}
	return dbWrapper
}

// seedUser inserts a user directly via the database layer and returns it (with ID populated).
func seedUser(t *testing.T, db *database.DB, email string, isAdmin bool, freeMonths int) *database.User {
	t.Helper()
	user := &database.User{
		GoogleID:  "google-" + email,
		Email:     email,
		Name:      "Test User " + email,
		CreatedAt: time.Now(),
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("failed to seed user %s: %v", email, err)
	}
	if isAdmin || freeMonths > 0 {
		if isAdmin {
			if err := db.SetUserAdmin(user.ID, isAdmin); err != nil {
				t.Fatalf("failed to set admin status for seeded user: %v", err)
			}
			user.IsAdmin = isAdmin
		}
		if freeMonths > 0 {
			if err := db.GrantFreeMonths(user.ID, freeMonths); err != nil {
				t.Fatalf("failed to grant free months for seeded user: %v", err)
			}
			user.FreeMonthsRemaining = freeMonths
		}
	}
	return user
}

// captureStdout redirects os.Stdout for the duration of fn and returns everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// runFatalSubprocess re-execs the current test binary running only the named test, in an
// environment variable-guarded child process. This lets us assert on log.Fatal/os.Exit(1)
// paths without killing the actual test process. See https://pkg.go.dev/os/exec#Cmd for the
// pattern (also used by the Go standard library's own tests).
func runFatalSubprocess(t *testing.T, testName string) (exitedNonZero bool, output string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^"+testName+"$", "-test.v")
	cmd.Env = append(os.Environ(), "GO_WANT_FATAL_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		return false, string(out)
	}
	if _, ok := err.(*exec.ExitError); ok {
		return true, string(out)
	}
	t.Fatalf("unexpected error running subprocess: %v", err)
	return false, string(out)
}

func isFatalSubprocess() bool {
	return os.Getenv("GO_WANT_FATAL_SUBPROCESS") == "1"
}

// mockStripeSubscriptionBackend points the Stripe SDK's API backend at a local httptest
// server that returns an "active" subscription for the given user ID on every request,
// avoiding live network calls to Stripe. Mirrors the pattern in
// internal/services/payment_service_test.go. Returns a cleanup func that restores the
// default backend.
func mockStripeSubscriptionBackend(t *testing.T, userID int) func() {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sub := stripe.Subscription{
			Status:             stripe.SubscriptionStatusActive,
			Metadata:           map[string]string{"user_id": fmt.Sprintf("%d", userID)},
			CurrentPeriodStart: time.Now().Unix(),
			CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0).Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sub)
	}))
	t.Cleanup(server.Close)

	apiBackend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
		URL:        stripe.String(server.URL),
		HTTPClient: server.Client(),
	})
	stripe.SetBackend(stripe.APIBackend, apiBackend)
	return func() {
		stripe.SetBackend(stripe.APIBackend, stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{}))
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"shorter than max", "abc", 10, "abc"},
		{"equal to max", "abcde", 5, "abcde"},
		{"longer than max", "abcdefghij", 5, "ab..."},
		{"empty string", "", 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncate(tt.in, tt.max); got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
		})
	}
}

func TestListUsers_SQLite(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	subscriptionService := services.NewSubscriptionService(db)

	t.Run("no users", func(t *testing.T) {
		output := captureStdout(t, func() { listUsers(db, subscriptionService) })
		if !bytes.Contains([]byte(output), []byte("Email")) {
			t.Errorf("expected header row in output, got: %s", output)
		}
	})

	t.Run("with users", func(t *testing.T) {
		seedUser(t, db, "alice@example.com", true, 2)
		seedUser(t, db, "bob@example.com", false, 0)

		output := captureStdout(t, func() { listUsers(db, subscriptionService) })
		if !bytes.Contains([]byte(output), []byte("alice@example.com")) {
			t.Errorf("expected alice in output, got: %s", output)
		}
		if !bytes.Contains([]byte(output), []byte("bob@example.com")) {
			t.Errorf("expected bob in output, got: %s", output)
		}
	})
}

func TestListUsers_UnsupportedDB(t *testing.T) {
	if isFatalSubprocess() {
		listUsers(nil, services.NewSubscriptionService(setupTestDB(t)))
		return
	}
	exited, output := runFatalSubprocess(t, "TestListUsers_UnsupportedDB")
	if !exited {
		t.Fatalf("expected listUsers to exit non-zero for an unsupported database type, output: %s", output)
	}
}

func TestSetAdminStatus(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	subscriptionService := services.NewSubscriptionService(db)
	auditService := services.NewAuditService(db)

	user := seedUser(t, db, "grant-me@example.com", false, 0)

	t.Run("grant admin", func(t *testing.T) {
		output := captureStdout(t, func() {
			setAdminStatus(subscriptionService, auditService, user.Email, true)
		})
		if !bytes.Contains([]byte(output), []byte("granted")) {
			t.Errorf("expected grant confirmation, got: %s", output)
		}
		got, err := subscriptionService.GetUserByEmail(user.Email)
		if err != nil {
			t.Fatalf("failed to reload user: %v", err)
		}
		if !got.IsAdmin {
			t.Errorf("expected user to be admin after grant")
		}
	})

	t.Run("revoke admin", func(t *testing.T) {
		output := captureStdout(t, func() {
			setAdminStatus(subscriptionService, auditService, user.Email, false)
		})
		if !bytes.Contains([]byte(output), []byte("removed from")) {
			t.Errorf("expected revoke confirmation, got: %s", output)
		}
		got, err := subscriptionService.GetUserByEmail(user.Email)
		if err != nil {
			t.Fatalf("failed to reload user: %v", err)
		}
		if got.IsAdmin {
			t.Errorf("expected user to not be admin after revoke")
		}
	})
}

func TestSetAdminStatus_UserNotFound(t *testing.T) {
	if isFatalSubprocess() {
		db := setupTestDB(t)
		defer func() { _ = db.Close() }()
		setAdminStatus(services.NewSubscriptionService(db), services.NewAuditService(db), "nobody@example.com", true)
		return
	}
	exited, output := runFatalSubprocess(t, "TestSetAdminStatus_UserNotFound")
	if !exited {
		t.Fatalf("expected setAdminStatus to exit non-zero for a missing user, output: %s", output)
	}
}

func TestGrantFreeMonths(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	subscriptionService := services.NewSubscriptionService(db)
	auditService := services.NewAuditService(db)

	user := seedUser(t, db, "trial@example.com", false, 1)

	output := captureStdout(t, func() {
		grantFreeMonths(subscriptionService, auditService, user.Email, 3)
	})
	if !bytes.Contains([]byte(output), []byte("Granted 3 free months")) {
		t.Errorf("expected grant confirmation, got: %s", output)
	}

	got, err := subscriptionService.GetUserByEmail(user.Email)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if got.FreeMonthsRemaining != 4 {
		t.Errorf("expected 4 free months remaining, got %d", got.FreeMonthsRemaining)
	}
}

func TestGrantFreeMonths_UserNotFound(t *testing.T) {
	if isFatalSubprocess() {
		db := setupTestDB(t)
		defer func() { _ = db.Close() }()
		grantFreeMonths(services.NewSubscriptionService(db), services.NewAuditService(db), "nobody@example.com", 1)
		return
	}
	exited, output := runFatalSubprocess(t, "TestGrantFreeMonths_UserNotFound")
	if !exited {
		t.Fatalf("expected grantFreeMonths to exit non-zero for a missing user, output: %s", output)
	}
}

func TestShowUserInfo(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	subscriptionService := services.NewSubscriptionService(db)

	user := seedUser(t, db, "info@example.com", true, 0)

	output := captureStdout(t, func() { showUserInfo(subscriptionService, user.Email) })
	if !bytes.Contains([]byte(output), []byte("info@example.com")) {
		t.Errorf("expected email in output, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("Is Admin")) {
		t.Errorf("expected admin field in output, got: %s", output)
	}
}

func TestShowUserInfo_UserNotFound(t *testing.T) {
	if isFatalSubprocess() {
		db := setupTestDB(t)
		defer func() { _ = db.Close() }()
		showUserInfo(services.NewSubscriptionService(db), "nobody@example.com")
		return
	}
	exited, output := runFatalSubprocess(t, "TestShowUserInfo_UserNotFound")
	if !exited {
		t.Fatalf("expected showUserInfo to exit non-zero for a missing user, output: %s", output)
	}
}

func TestFixSubscriptionStatus_NoSubscriptionID(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	subscriptionService := services.NewSubscriptionService(db)

	user := seedUser(t, db, "nosub@example.com", false, 0)

	// User has no SubscriptionID, so this must return early without contacting Stripe.
	output := captureStdout(t, func() { fixSubscriptionStatus(subscriptionService, user.Email) })
	if !bytes.Contains([]byte(output), []byte("nothing to fix")) {
		t.Errorf("expected early-return message, got: %s", output)
	}
}

func TestFixSubscriptionStatus_UserNotFound(t *testing.T) {
	if isFatalSubprocess() {
		db := setupTestDB(t)
		defer func() { _ = db.Close() }()
		fixSubscriptionStatus(services.NewSubscriptionService(db), "nobody@example.com")
		return
	}
	exited, output := runFatalSubprocess(t, "TestFixSubscriptionStatus_UserNotFound")
	if !exited {
		t.Fatalf("expected fixSubscriptionStatus to exit non-zero for a missing user, output: %s", output)
	}
}

func TestFixSubscriptionStatus_WithSubscriptionID(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	subscriptionService := services.NewSubscriptionService(db)

	user := seedUser(t, db, "hassub@example.com", false, 0)
	if err := subscriptionService.UpdateUserSubscription(user.ID, "trial", "sub_123", time.Time{}, time.Time{}); err != nil {
		t.Fatalf("failed to set subscription id on seeded user: %v", err)
	}

	restore := mockStripeSubscriptionBackend(t, user.ID)
	defer restore()

	output := captureStdout(t, func() { fixSubscriptionStatus(subscriptionService, user.Email) })
	if !bytes.Contains([]byte(output), []byte("Subscription status updated successfully")) {
		t.Errorf("expected success message, got: %s", output)
	}

	got, err := subscriptionService.GetUserByEmail(user.Email)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if got.SubscriptionStatus != "active" {
		t.Errorf("expected subscription status 'active' after sync, got %q", got.SubscriptionStatus)
	}
}

func TestSetSubscriptionID(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	subscriptionService := services.NewSubscriptionService(db)

	user := seedUser(t, db, "setsub@example.com", false, 0)

	restore := mockStripeSubscriptionBackend(t, user.ID)
	defer restore()

	output := captureStdout(t, func() { setSubscriptionID(subscriptionService, user.Email, "sub_new456") })
	if !bytes.Contains([]byte(output), []byte("Subscription ID updated successfully")) {
		t.Errorf("expected update confirmation, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("Status synced from Stripe: active")) {
		t.Errorf("expected Stripe sync confirmation, got: %s", output)
	}

	got, err := subscriptionService.GetUserByEmail(user.Email)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if got.SubscriptionID != "sub_new456" {
		t.Errorf("expected subscription id 'sub_new456', got %q", got.SubscriptionID)
	}
	if got.SubscriptionStatus != "active" {
		t.Errorf("expected subscription status 'active' after sync, got %q", got.SubscriptionStatus)
	}
}

func TestSetSubscriptionID_UserNotFound(t *testing.T) {
	if isFatalSubprocess() {
		db := setupTestDB(t)
		defer func() { _ = db.Close() }()
		setSubscriptionID(services.NewSubscriptionService(db), "nobody@example.com", "sub_x")
		return
	}
	exited, output := runFatalSubprocess(t, "TestSetSubscriptionID_UserNotFound")
	if !exited {
		t.Fatalf("expected setSubscriptionID to exit non-zero for a missing user, output: %s", output)
	}
}

func TestListAdminTokens(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	subscriptionService := services.NewSubscriptionService(db)

	t.Run("no tokens", func(t *testing.T) {
		output := captureStdout(t, func() { listAdminTokens(subscriptionService) })
		if !bytes.Contains([]byte(output), []byte("No admin tokens found")) {
			t.Errorf("expected empty-state message, got: %s", output)
		}
	})

	t.Run("with tokens", func(t *testing.T) {
		if _, err := subscriptionService.GenerateAdminToken("first token"); err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		output := captureStdout(t, func() { listAdminTokens(subscriptionService) })
		if !bytes.Contains([]byte(output), []byte("first token")) {
			t.Errorf("expected token description in output, got: %s", output)
		}
		if !bytes.Contains([]byte(output), []byte("Active")) {
			t.Errorf("expected Active status in output, got: %s", output)
		}
	})
}

func TestRevokeAdminToken(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	subscriptionService := services.NewSubscriptionService(db)

	token, err := subscriptionService.GenerateAdminToken("to be revoked")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	tokens, err := subscriptionService.ListAdminTokens()
	if err != nil || len(tokens) != 1 {
		t.Fatalf("expected exactly one token, got %d tokens, err=%v", len(tokens), err)
	}

	output := captureStdout(t, func() { revokeAdminToken(subscriptionService, tokens[0].ID) })
	if !bytes.Contains([]byte(output), []byte("has been revoked")) {
		t.Errorf("expected revocation confirmation, got: %s", output)
	}

	valid, err := subscriptionService.ValidateAdminToken(token)
	if err != nil {
		t.Fatalf("unexpected error validating revoked token: %v", err)
	}
	if valid {
		t.Errorf("expected revoked token to no longer validate")
	}
}

func TestRevokeAdminToken_NotFound(t *testing.T) {
	if isFatalSubprocess() {
		db := setupTestDB(t)
		defer func() { _ = db.Close() }()
		revokeAdminToken(services.NewSubscriptionService(db), 999)
		return
	}
	exited, output := runFatalSubprocess(t, "TestRevokeAdminToken_NotFound")
	if !exited {
		t.Fatalf("expected revokeAdminToken to exit non-zero for an unknown token id, output: %s", output)
	}
}

func TestShowAuditLogs(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	auditService := services.NewAuditService(db)

	t.Run("no logs", func(t *testing.T) {
		output := captureStdout(t, func() { showAuditLogs(auditService, []string{}) })
		if !bytes.Contains([]byte(output), []byte("No audit logs found")) {
			t.Errorf("expected empty-state message, got: %s", output)
		}
	})

	t.Run("with logs and args", func(t *testing.T) {
		if err := auditService.LogSuccess(1, "CLI_ADMIN", "grant_admin", 2, "target@example.com", nil, "CLI"); err != nil {
			t.Fatalf("failed to write audit log: %v", err)
		}

		output := captureStdout(t, func() {
			showAuditLogs(auditService, []string{"--limit", "5", "--operation", "grant_admin"})
		})
		if !bytes.Contains([]byte(output), []byte("target@example.com")) {
			t.Errorf("expected target email in output, got: %s", output)
		}
		if !bytes.Contains([]byte(output), []byte("grant_admin")) {
			t.Errorf("expected operation type in output, got: %s", output)
		}
	})
}

// TestHasAdminUsers exercises the standalone hasAdminUsers helper, which opens its own
// database.InitDB() connection rather than reusing an injected one. Chdir into a scratch
// directory so it operates on a throwaway SQLite file instead of the repo's working directory.
func TestHasAdminUsers(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Chdir(t.TempDir())

	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	sqliteDB, ok := db.(*database.DB)
	if !ok {
		t.Fatalf("expected *database.DB, got %T", db)
	}
	subscriptionService := services.NewSubscriptionService(sqliteDB)

	if hasAdminUsers(subscriptionService) {
		t.Errorf("expected no admin users in a fresh database")
	}

	seedUser(t, sqliteDB, "admin@example.com", true, 0)
	_ = sqliteDB.Close()

	if !hasAdminUsers(subscriptionService) {
		t.Errorf("expected an admin user to be found after seeding one")
	}
}

// TestCreateAdminToken_Bootstrap exercises createAdminToken's first-token bootstrap path,
// which requires an existing admin user and no existing tokens. Like TestHasAdminUsers, it
// chdirs into a scratch directory because createAdminToken calls hasAdminUsers, which opens
// its own database.InitDB() connection rather than reusing the injected service's db.
func TestCreateAdminToken_Bootstrap(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Chdir(t.TempDir())

	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	sqliteDB, ok := db.(*database.DB)
	if !ok {
		t.Fatalf("expected *database.DB, got %T", db)
	}
	defer func() { _ = sqliteDB.Close() }()

	seedUser(t, sqliteDB, "bootstrap-admin@example.com", true, 0)
	subscriptionService := services.NewSubscriptionService(sqliteDB)

	output := captureStdout(t, func() { createAdminToken(subscriptionService, "bootstrap token") })
	if !bytes.Contains([]byte(output), []byte("Admin token created successfully")) {
		t.Errorf("expected success message, got: %s", output)
	}

	tokens, err := subscriptionService.ListAdminTokens()
	if err != nil {
		t.Fatalf("failed to list admin tokens: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Description != "bootstrap token" {
		t.Errorf("expected exactly one token with description 'bootstrap token', got: %+v", tokens)
	}
}

func TestCreateAdminToken_NoAdminUsers(t *testing.T) {
	if isFatalSubprocess() {
		t.Setenv("GOOGLE_CLOUD_PROJECT", "")
		t.Chdir(t.TempDir())

		db, err := database.InitDB()
		if err != nil {
			t.Fatalf("failed to init db: %v", err)
		}
		sqliteDB := db.(*database.DB)
		defer func() { _ = sqliteDB.Close() }()

		createAdminToken(services.NewSubscriptionService(sqliteDB), "should never be created")
		return
	}
	exited, output := runFatalSubprocess(t, "TestCreateAdminToken_NoAdminUsers")
	if !exited {
		t.Fatalf("expected createAdminToken to exit non-zero with no admin users, output: %s", output)
	}
}
