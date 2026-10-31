package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/secrets"
	"github.com/jeffreyp/goread2/test/helpers"
)

// TestSignInAfterAccountDeletionStartsWithTrialEnded drives the anti-abuse path
// end to end: DELETE /api/account writes a tombstone for the Google ID, and
// signing in again with that Google ID creates a new account whose trial ended
// at creation, so it is not subscription-active. A Google ID with no deleted
// account still gets the 30-day trial. Each case runs on SQLite and on the
// Datastore emulator, since both CreateUser implementations must keep a
// preset TrialEndsAt instead of applying the 30-day default.
func TestSignInAfterAccountDeletionStartsWithTrialEnded(t *testing.T) {
	// The tombstone ID is an HMAC of the Google ID under this key; without a
	// key, deletion skips the tombstone and the check under test never runs.
	t.Setenv("ACCOUNT_TOMBSTONE_KEY", "integration-test-tombstone-key")
	secrets.ResetCacheForTesting()
	t.Cleanup(secrets.ResetCacheForTesting)

	backends := []struct {
		name  string
		newDB func(t *testing.T) database.Database
	}{
		{"SQLite", func(t *testing.T) database.Database { return helpers.CreateTestDB(t) }},
		{"Datastore", helpers.CreateTestDatastoreDB},
	}

	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			ts := helpers.SetupTestServerWithDB(t, backend.newDB(t))
			ctx := context.Background()

			const googleID = "google-deleted-then-returning"
			const email = "returning@example.com"

			original, err := ts.AuthService.SignInGoogleUser(ctx, auth.GoogleUserInfo{ID: googleID, Email: email, Name: "Returning User"})
			if err != nil {
				t.Fatalf("First sign-in failed: %v", err)
			}
			assertTrialDays(t, ts.DB, original.ID, 30)

			req := ts.CreateAuthenticatedRequest(t, http.MethodDelete, "/api/account", map[string]string{"confirm_email": email}, original)
			if rr := ts.ExecuteRequest(req); rr.Code != http.StatusOK {
				t.Fatalf("DELETE /api/account returned %d: %s", rr.Code, rr.Body.String())
			}
			if _, err := ts.DB.GetUserByGoogleID(googleID); err == nil {
				t.Fatal("Expected the user to be gone after DELETE /api/account")
			}

			returning, err := ts.AuthService.SignInGoogleUser(ctx, auth.GoogleUserInfo{ID: googleID, Email: email, Name: "Returning User"})
			if err != nil {
				t.Fatalf("Sign-in after deletion failed: %v", err)
			}
			if returning.ID == original.ID {
				t.Fatalf("Expected a new user ID after deletion, got the original ID %d again", original.ID)
			}
			assertTrialDays(t, ts.DB, returning.ID, 0)

			active, err := ts.DB.IsUserSubscriptionActive(returning.ID)
			if err != nil {
				t.Fatalf("IsUserSubscriptionActive failed: %v", err)
			}
			if active {
				t.Error("Expected the re-created account not to be subscription-active, since its trial has already ended")
			}

			// Control: a Google ID that never deleted an account is unaffected.
			fresh, err := ts.AuthService.SignInGoogleUser(ctx, auth.GoogleUserInfo{ID: "google-never-deleted", Email: "fresh@example.com", Name: "Fresh User"})
			if err != nil {
				t.Fatalf("Sign-in for a new Google ID failed: %v", err)
			}
			assertTrialDays(t, ts.DB, fresh.ID, 30)
			if active, err := ts.DB.IsUserSubscriptionActive(fresh.ID); err != nil || !active {
				t.Errorf("Expected a never-deleted account to be subscription-active on trial, got active=%v, err=%v", active, err)
			}
		})
	}
}

// assertTrialDays reloads userID from the database and checks that its trial
// ends the given number of days after CreatedAt and that it is on the trial
// status. Reloading verifies what CreateUser stored, not the in-memory struct.
func assertTrialDays(t *testing.T, db database.Database, userID, days int) {
	t.Helper()
	stored, err := db.GetUserByID(userID)
	if err != nil {
		t.Fatalf("GetUserByID(%d) failed: %v", userID, err)
	}
	if stored.SubscriptionStatus != "trial" {
		t.Errorf("User %d: expected subscription status %q, got %q", userID, "trial", stored.SubscriptionStatus)
	}
	// CreateUser adds the 30 days to CreatedAt in the local zone, but Datastore
	// returns times in UTC, so a span that crosses a DST change would differ by
	// an hour if the days were added in UTC.
	want := stored.CreatedAt.In(time.Local).AddDate(0, 0, days)
	if diff := stored.TrialEndsAt.Sub(want); diff < -time.Second || diff > time.Second {
		t.Errorf("User %d: expected TrialEndsAt %v (CreatedAt + %d days), got %v", userID, want, days, stored.TrialEndsAt)
	}
}
