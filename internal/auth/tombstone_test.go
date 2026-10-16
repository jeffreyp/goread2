package auth

import (
	"context"
	"testing"

	"github.com/jeffreyp/goread2/internal/secrets"
)

// tombstoneDB reports a tombstone for exactly one ID.
type tombstoneDB struct {
	*mockDB
	tombstoneID string
}

func (db *tombstoneDB) HasAccountTombstone(id string) (bool, error) {
	return id == db.tombstoneID, nil
}

func TestHasDeletedAccount(t *testing.T) {
	t.Setenv("ACCOUNT_TOMBSTONE_KEY", "test-tombstone-key")
	secrets.ResetCacheForTesting()
	t.Cleanup(secrets.ResetCacheForTesting)

	deletedID, err := secrets.AccountTombstoneID(context.Background(), "google-deleted")
	if err != nil || deletedID == "" {
		t.Fatalf("Failed to compute tombstone ID: %q, %v", deletedID, err)
	}
	a := &AuthService{db: &tombstoneDB{mockDB: newMockDB(), tombstoneID: deletedID}}

	if !a.hasDeletedAccount(context.Background(), "google-deleted") {
		t.Error("Expected a tombstoned Google ID to be reported as deleted")
	}
	if a.hasDeletedAccount(context.Background(), "google-new") {
		t.Error("Expected an unknown Google ID not to be reported as deleted")
	}
}

func TestAccountTombstoneIDIsKeyed(t *testing.T) {
	t.Setenv("ACCOUNT_TOMBSTONE_KEY", "key-one")
	secrets.ResetCacheForTesting()
	t.Cleanup(secrets.ResetCacheForTesting)
	first, _ := secrets.AccountTombstoneID(context.Background(), "google-123")

	t.Setenv("ACCOUNT_TOMBSTONE_KEY", "key-two")
	secrets.ResetCacheForTesting()
	second, _ := secrets.AccountTombstoneID(context.Background(), "google-123")

	if first == "" || first == second {
		t.Errorf("Expected different non-empty IDs under different keys, got %q and %q", first, second)
	}
}
