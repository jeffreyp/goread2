package services

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/secrets"
)

// mockDBDeletion records the order of the database calls DeleteAccount makes.
// It embeds database.Database so that any call it does not override panics,
// which fails the test if DeleteAccount starts touching other data.
type mockDBDeletion struct {
	database.Database
	user  *database.User
	calls *[]string
}

func (m *mockDBDeletion) GetUserByID(userID int) (*database.User, error) {
	*m.calls = append(*m.calls, "GetUserByID")
	if m.user == nil {
		return nil, database.ErrUserNotFound
	}
	return m.user, nil
}

func (m *mockDBDeletion) CreateAccountTombstone(id string) error {
	*m.calls = append(*m.calls, "CreateAccountTombstone")
	return nil
}

func (m *mockDBDeletion) DeleteUserAccount(userID int) error {
	*m.calls = append(*m.calls, "DeleteUserAccount")
	return nil
}

type fakeCustomerDeleter struct {
	err   error
	calls *[]string
}

func (f *fakeCustomerDeleter) DeleteCustomerForUser(user *database.User) error {
	*f.calls = append(*f.calls, "DeleteCustomerForUser")
	return f.err
}

func (f *fakeCustomerDeleter) DeleteCustomersByUserID(userID int) error {
	*f.calls = append(*f.calls, "DeleteCustomersByUserID")
	return f.err
}

// setTombstoneKey configures the account tombstone key for one test. An empty
// key disables tombstones; GOOGLE_CLOUD_PROJECT is cleared so the lookup does
// not fall through to Secret Manager.
func setTombstoneKey(t *testing.T, key string) {
	t.Helper()
	t.Setenv("ACCOUNT_TOMBSTONE_KEY", key)
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	secrets.ResetCacheForTesting()
	t.Cleanup(secrets.ResetCacheForTesting)
}

func TestDeleteAccount(t *testing.T) {
	user := &database.User{ID: 7, GoogleID: "google-7", Email: "user@example.com"}
	stripeErr := errors.New("stripe unavailable")

	tests := []struct {
		name         string
		user         *database.User
		tombstoneKey string
		stripeErr    error
		wantErr      string
		wantCalls    []string
	}{
		{
			name:         "Stripe customer is deleted before the tombstone and the data",
			user:         user,
			tombstoneKey: "test-key",
			wantCalls:    []string{"GetUserByID", "DeleteCustomerForUser", "CreateAccountTombstone", "DeleteUserAccount"},
		},
		{
			name:         "Stripe failure aborts before any tombstone or data deletion",
			user:         user,
			tombstoneKey: "test-key",
			stripeErr:    stripeErr,
			wantErr:      "failed to delete Stripe customer",
			wantCalls:    []string{"GetUserByID", "DeleteCustomerForUser"},
		},
		{
			name:         "sweep after the user is gone deletes Stripe customers by user ID",
			user:         nil,
			tombstoneKey: "test-key",
			wantCalls:    []string{"GetUserByID", "DeleteCustomersByUserID", "DeleteUserAccount"},
		},
		{
			name:         "sweep Stripe failure aborts before data deletion",
			user:         nil,
			tombstoneKey: "test-key",
			stripeErr:    stripeErr,
			wantErr:      "failed to delete Stripe customers",
			wantCalls:    []string{"GetUserByID", "DeleteCustomersByUserID"},
		},
		{
			name:         "missing tombstone key deletes the account without a tombstone",
			user:         user,
			tombstoneKey: "",
			wantCalls:    []string{"GetUserByID", "DeleteCustomerForUser", "DeleteUserAccount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTombstoneKey(t, tt.tombstoneKey)

			var calls []string
			service := NewUserDataService(&mockDBDeletion{user: tt.user, calls: &calls})
			service.SetCustomerDeleter(&fakeCustomerDeleter{err: tt.stripeErr, calls: &calls})

			err := service.DeleteAccount(user.ID)

			if tt.wantErr == "" && err != nil {
				t.Fatalf("DeleteAccount() error = %v, want nil", err)
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("DeleteAccount() error = %v, want error containing %q", err, tt.wantErr)
				}
				if tt.stripeErr != nil && !errors.Is(err, tt.stripeErr) {
					t.Errorf("DeleteAccount() error = %v, want it to wrap %v", err, tt.stripeErr)
				}
			}
			if !reflect.DeepEqual(calls, tt.wantCalls) {
				t.Errorf("calls = %v, want %v", calls, tt.wantCalls)
			}
		})
	}
}

func TestDeleteAccountWithoutCustomerDeleter(t *testing.T) {
	tests := []struct {
		name      string
		user      *database.User
		wantErr   bool
		wantCalls []string
	}{
		{
			name:      "live subscription is refused so Stripe does not keep billing",
			user:      &database.User{ID: 7, GoogleID: "google-7", SubscriptionID: "sub_1", SubscriptionStatus: "active"},
			wantErr:   true,
			wantCalls: []string{"GetUserByID"},
		},
		{
			name:      "past_due subscription is refused",
			user:      &database.User{ID: 7, GoogleID: "google-7", SubscriptionID: "sub_1", SubscriptionStatus: "past_due"},
			wantErr:   true,
			wantCalls: []string{"GetUserByID"},
		},
		{
			name:      "cancelled subscription is deleted",
			user:      &database.User{ID: 7, GoogleID: "google-7", SubscriptionID: "sub_1", SubscriptionStatus: "cancelled"},
			wantCalls: []string{"GetUserByID", "CreateAccountTombstone", "DeleteUserAccount"},
		},
		{
			name:      "trial user is deleted",
			user:      &database.User{ID: 7, GoogleID: "google-7", SubscriptionStatus: "trial"},
			wantCalls: []string{"GetUserByID", "CreateAccountTombstone", "DeleteUserAccount"},
		},
		{
			name:      "sweep after the user is gone deletes data",
			user:      nil,
			wantCalls: []string{"GetUserByID", "DeleteUserAccount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTombstoneKey(t, "test-key")

			var calls []string
			service := NewUserDataService(&mockDBDeletion{user: tt.user, calls: &calls})

			err := service.DeleteAccount(7)

			if tt.wantErr != errors.Is(err, ErrNoCustomerDeleter) {
				t.Fatalf("DeleteAccount() error = %v, want ErrNoCustomerDeleter: %v", err, tt.wantErr)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("DeleteAccount() error = %v, want nil", err)
			}
			if !reflect.DeepEqual(calls, tt.wantCalls) {
				t.Errorf("calls = %v, want %v", calls, tt.wantCalls)
			}
		})
	}
}
