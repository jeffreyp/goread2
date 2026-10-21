package services

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

func TestDecideRetention(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	longAgo := now.AddDate(0, -24, -1) // just past the 24-month cutoff
	recently := now.AddDate(0, -23, 0)

	tests := []struct {
		name string
		info database.UserRetentionInfo
		want retentionDecision
	}{
		{"inactive trial is deleted", database.UserRetentionInfo{SubscriptionStatus: "trial", LastSeenAt: longAgo}, deleteAccount},
		{"inactive cancelled is deleted", database.UserRetentionInfo{SubscriptionStatus: "cancelled", LastSeenAt: longAgo}, deleteAccount},
		{"inactive expired is deleted", database.UserRetentionInfo{SubscriptionStatus: "expired", LastSeenAt: longAgo}, deleteAccount},
		{"exactly 24 months is kept", database.UserRetentionInfo{SubscriptionStatus: "expired", LastSeenAt: now.AddDate(0, -24, 0)}, retainAccount},
		{"recently active is kept", database.UserRetentionInfo{SubscriptionStatus: "expired", LastSeenAt: recently}, retainAccount},
		{"active subscriber is kept", database.UserRetentionInfo{SubscriptionStatus: "active", LastSeenAt: longAgo}, retainAccount},
		{"past due is kept", database.UserRetentionInfo{SubscriptionStatus: "past_due", LastSeenAt: longAgo}, retainAccount},
		{"unpaid is kept", database.UserRetentionInfo{SubscriptionStatus: "unpaid", LastSeenAt: longAgo}, retainAccount},
		{"unknown status is kept", database.UserRetentionInfo{SubscriptionStatus: "", LastSeenAt: longAgo}, retainAccount},
		{"admin is kept", database.UserRetentionInfo{SubscriptionStatus: "expired", IsAdmin: true, LastSeenAt: longAgo}, retainAccount},
		{"granted free months is kept", database.UserRetentionInfo{SubscriptionStatus: "expired", FreeMonthsRemaining: 2, LastSeenAt: longAgo}, retainAccount},
		{"no recorded activity is backfilled", database.UserRetentionInfo{SubscriptionStatus: "expired", CreatedAt: longAgo}, backfillActivity},
		{"no activity but paying is kept", database.UserRetentionInfo{SubscriptionStatus: "active"}, retainAccount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decideRetention(tt.info, now); got != tt.want {
				t.Errorf("decideRetention(%+v) = %v, want %v", tt.info, got, tt.want)
			}
		})
	}
}

// mockDBRetention serves Run's account list and records what Run does with
// it. It embeds database.Database so that any call it does not override
// panics. GetUserByID reports every user as gone, so DeleteAccount goes
// straight to DeleteUserAccount.
type mockDBRetention struct {
	database.Database
	infos         []database.UserRetentionInfo
	infoErr       error
	failDeleteFor int
	deletedIDs    []int
	backfilled    map[int]time.Time
	clearCutoff   time.Time
	clearCalls    int
	ipsCleared    int
}

func (m *mockDBRetention) GetUserRetentionInfo() ([]database.UserRetentionInfo, error) {
	return m.infos, m.infoErr
}

func (m *mockDBRetention) RecordUserActivity(userID int, at time.Time) error {
	if m.backfilled == nil {
		m.backfilled = make(map[int]time.Time)
	}
	m.backfilled[userID] = at
	return nil
}

func (m *mockDBRetention) GetUserByID(int) (*database.User, error) {
	return nil, database.ErrUserNotFound
}

func (m *mockDBRetention) DeleteUserAccount(userID int) error {
	if userID == m.failDeleteFor {
		return errors.New("datastore unavailable")
	}
	m.deletedIDs = append(m.deletedIDs, userID)
	return nil
}

func (m *mockDBRetention) ClearAuditLogIPsBefore(cutoff time.Time) (int, error) {
	m.clearCalls++
	m.clearCutoff = cutoff
	return m.ipsCleared, nil
}

func TestRetentionRunContinuesPastFailedDeletion(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	longAgo := now.AddDate(0, -25, 0)
	db := &mockDBRetention{
		infos: []database.UserRetentionInfo{
			{UserID: 1, SubscriptionStatus: "expired", LastSeenAt: longAgo},
			{UserID: 2, SubscriptionStatus: "trial", LastSeenAt: longAgo},
			{UserID: 3, SubscriptionStatus: "cancelled", LastSeenAt: longAgo},
			{UserID: 4, SubscriptionStatus: "active", LastSeenAt: longAgo},
			{UserID: 5, SubscriptionStatus: "expired", CreatedAt: longAgo},
		},
		failDeleteFor: 2,
		ipsCleared:    6,
	}
	service := NewRetentionService(db, NewUserDataService(db))

	result, err := service.Run(now)

	if err == nil || !strings.Contains(err.Error(), "1 account deletions failed") {
		t.Errorf("Run() error = %v, want the failed deletion reported so the task retries", err)
	}
	want := RetentionResult{AccountsDeleted: 2, AccountsBackfilled: 1, DeletionsFailed: 1, AuditIPsCleared: 6}
	if result != want {
		t.Errorf("Run() result = %+v, want %+v", result, want)
	}
	if !reflect.DeepEqual(db.deletedIDs, []int{1, 3}) {
		t.Errorf("deleted users = %v, want [1 3]", db.deletedIDs)
	}
	if !reflect.DeepEqual(db.backfilled, map[int]time.Time{5: now}) {
		t.Errorf("backfilled = %v, want user 5 at %v", db.backfilled, now)
	}
	if db.clearCalls != 1 || !db.clearCutoff.Equal(now.AddDate(0, -auditLogIPRetentionMonths, 0)) {
		t.Errorf("ClearAuditLogIPsBefore called %d times with cutoff %v, want once with %v",
			db.clearCalls, db.clearCutoff, now.AddDate(0, -auditLogIPRetentionMonths, 0))
	}
}

func TestRetentionRunStopsWhenAccountsCannotBeLoaded(t *testing.T) {
	db := &mockDBRetention{infoErr: errors.New("datastore unavailable")}
	service := NewRetentionService(db, NewUserDataService(db))

	result, err := service.Run(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC))

	if err == nil || !strings.Contains(err.Error(), "failed to load accounts") {
		t.Errorf("Run() error = %v, want failure to load accounts", err)
	}
	if result != (RetentionResult{}) {
		t.Errorf("Run() result = %+v, want zero counts", result)
	}
	if db.clearCalls != 0 {
		t.Errorf("ClearAuditLogIPsBefore called %d times, want 0", db.clearCalls)
	}
}
