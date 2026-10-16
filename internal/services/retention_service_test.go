package services

import (
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
