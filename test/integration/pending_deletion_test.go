package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/services"
	"github.com/jeffreyp/goread2/test/helpers"
)

// recordingCustomerDeleter stands in for Stripe, recording which users'
// customers DeleteAccount asked it to delete.
type recordingCustomerDeleter struct {
	deletedFor []int
}

func (d *recordingCustomerDeleter) DeleteCustomerForUser(user *database.User) error {
	d.deletedFor = append(d.deletedFor, user.ID)
	return nil
}

func (d *recordingCustomerDeleter) DeleteCustomersByUserID(int) error { return nil }

// TestPendingDeletionLifecycle drives the deletion-retry marker through each
// database backend: RequestDeletion stores deletion_requested_at, every User
// loader and GetUserRetentionInfo return it, SignInGoogleUser refuses the
// account while it is set, RetentionService.Run leaves a fresh request to
// Cloud Tasks but retries a stale one even for an active subscriber, and once
// the account is gone the same Google identity can sign in to a new account.
func TestPendingDeletionLifecycle(t *testing.T) {
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
			identity := auth.GoogleUserInfo{ID: "google-pending-deletion", Email: "pending@example.com", Name: "Pending User"}

			user, err := ts.AuthService.SignInGoogleUser(ctx, identity)
			if err != nil {
				t.Fatalf("First sign-in failed: %v", err)
			}
			if err := ts.DB.UpdateUserSubscription(user.ID, "active", "sub_pending", time.Now(), time.Now().AddDate(0, 1, 0)); err != nil {
				t.Fatalf("UpdateUserSubscription failed: %v", err)
			}

			stripe := &recordingCustomerDeleter{}
			userData := services.NewUserDataService(ts.DB)
			userData.SetCustomerDeleter(stripe)

			if err := userData.RequestDeletion(user.ID); err != nil {
				t.Fatalf("RequestDeletion failed: %v", err)
			}
			requestedAt := assertDeletionRequested(t, ts.DB, user)

			// A later update to the user must not clear the marker.
			if err := ts.DB.UpdateUserMaxArticlesOnFeedAdd(user.ID, 50); err != nil {
				t.Fatalf("UpdateUserMaxArticlesOnFeedAdd failed: %v", err)
			}
			if stored, err := ts.DB.GetUserByID(user.ID); err != nil || stored.DeletionRequestedAt.IsZero() {
				t.Fatalf("Expected deletion_requested_at to survive another user update, got %+v, err=%v", stored, err)
			}

			if _, err := ts.AuthService.SignInGoogleUser(ctx, identity); !errors.Is(err, auth.ErrAccountDeletionPending) {
				t.Fatalf("Sign-in with a deletion pending returned err=%v, want ErrAccountDeletionPending", err)
			}

			retention := services.NewRetentionService(ts.DB, userData)

			// Within the retry delay, Cloud Tasks may still be working on the
			// deletion, and an active subscriber is otherwise never deleted.
			if result, err := retention.Run(requestedAt.Add(30 * time.Minute)); err != nil || result.AccountsDeleted != 0 {
				t.Fatalf("Run within the retry delay: result=%+v, err=%v; want nothing deleted", result, err)
			}
			if _, err := ts.DB.GetUserByID(user.ID); err != nil {
				t.Fatalf("Expected the user to survive a run within the retry delay: %v", err)
			}

			result, err := retention.Run(requestedAt.Add(2 * time.Hour))
			if err != nil {
				t.Fatalf("Run after the retry delay failed: %v", err)
			}
			if result.AccountsDeleted != 1 {
				t.Errorf("Run after the retry delay deleted %d accounts, want 1", result.AccountsDeleted)
			}
			if _, err := ts.DB.GetUserByID(user.ID); !errors.Is(err, database.ErrUserNotFound) {
				t.Fatalf("Expected the user to be gone after the retry, got err=%v", err)
			}
			if len(stripe.deletedFor) != 1 || stripe.deletedFor[0] != user.ID {
				t.Errorf("Stripe customer deletions = %v, want [%d]", stripe.deletedFor, user.ID)
			}

			returning, err := ts.AuthService.SignInGoogleUser(ctx, identity)
			if err != nil {
				t.Fatalf("Sign-in after the deletion completed failed: %v", err)
			}
			if returning.ID == user.ID || !returning.DeletionRequestedAt.IsZero() {
				t.Errorf("Expected a new account without a pending deletion, got ID %d (original %d), deletion_requested_at %v",
					returning.ID, user.ID, returning.DeletionRequestedAt)
			}
		})
	}
}

// assertDeletionRequested checks that every way of loading user reports a
// pending deletion, and returns the stored request time.
func assertDeletionRequested(t *testing.T, db database.Database, user *database.User) time.Time {
	t.Helper()

	byID, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	requestedAt := byID.DeletionRequestedAt
	if requestedAt.IsZero() || time.Since(requestedAt) > time.Minute {
		t.Fatalf("GetUserByID deletion_requested_at = %v, want close to now", requestedAt)
	}

	byGoogleID, err := db.GetUserByGoogleID(user.GoogleID)
	if err != nil {
		t.Fatalf("GetUserByGoogleID failed: %v", err)
	}
	if !byGoogleID.DeletionRequestedAt.Equal(requestedAt) {
		t.Errorf("GetUserByGoogleID deletion_requested_at = %v, want %v", byGoogleID.DeletionRequestedAt, requestedAt)
	}

	byEmail, err := db.GetUserByEmail(user.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if !byEmail.DeletionRequestedAt.Equal(requestedAt) {
		t.Errorf("GetUserByEmail deletion_requested_at = %v, want %v", byEmail.DeletionRequestedAt, requestedAt)
	}

	infos, err := db.GetUserRetentionInfo()
	if err != nil {
		t.Fatalf("GetUserRetentionInfo failed: %v", err)
	}
	found := false
	for _, info := range infos {
		if info.UserID == user.ID {
			found = true
			if !info.DeletionRequestedAt.Equal(requestedAt) {
				t.Errorf("GetUserRetentionInfo deletion_requested_at = %v, want %v", info.DeletionRequestedAt, requestedAt)
			}
		}
	}
	if !found {
		t.Errorf("GetUserRetentionInfo did not include user %d", user.ID)
	}

	return requestedAt
}
