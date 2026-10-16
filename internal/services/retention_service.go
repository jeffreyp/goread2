package services

import (
	"fmt"
	"log"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

const (
	// inactiveAccountRetentionMonths is how long an unpaid account may go
	// without any sign-in or activity before it is deleted.
	inactiveAccountRetentionMonths = 24
	// auditLogIPRetentionMonths is how long audit log entries keep the IP
	// address of the admin who performed the action.
	auditLogIPRetentionMonths = 12
)

// RetentionService applies GoRead2's data retention policy: it deletes
// inactive unpaid accounts and expires IP addresses in the audit log.
type RetentionService struct {
	db              database.Database
	userDataService *UserDataService
}

func NewRetentionService(db database.Database, userDataService *UserDataService) *RetentionService {
	return &RetentionService{db: db, userDataService: userDataService}
}

// RetentionResult summarizes one retention run.
type RetentionResult struct {
	AccountsDeleted    int `json:"accounts_deleted"`
	AccountsBackfilled int `json:"accounts_backfilled"`
	DeletionsFailed    int `json:"deletions_failed"`
	AuditIPsCleared    int `json:"audit_ips_cleared"`
}

// retentionDecision is what Run does with one account.
type retentionDecision int

const (
	retainAccount retentionDecision = iota
	backfillActivity
	deleteAccount
)

// decideRetention applies the inactive-account rule to one account. Accounts
// that are paying, admins, or holding granted free months are always kept.
// An account with no recorded activity predates activity tracking; it is
// treated as active now rather than judged by its creation date, which could
// delete long-standing users who were active before tracking began.
func decideRetention(info database.UserRetentionInfo, now time.Time) retentionDecision {
	switch info.SubscriptionStatus {
	case "trial", "cancelled", "expired":
	default:
		return retainAccount // active, past_due, unpaid, admin, or unknown
	}
	if info.IsAdmin || info.FreeMonthsRemaining > 0 {
		return retainAccount
	}
	if info.LastSeenAt.IsZero() {
		return backfillActivity
	}
	if info.LastSeenAt.Before(now.AddDate(0, -inactiveAccountRetentionMonths, 0)) {
		return deleteAccount
	}
	return retainAccount
}

// Run applies the retention policy as of now. A failure to delete one account
// is logged and counted, and does not stop the run; Run returns an error only
// if it cannot load the account list or expire audit log IPs, or if any
// deletion failed, so that the task is retried. Deletion is idempotent, so a
// retry re-attempts only the work still outstanding.
func (s *RetentionService) Run(now time.Time) (RetentionResult, error) {
	var result RetentionResult

	infos, err := s.db.GetUserRetentionInfo()
	if err != nil {
		return result, fmt.Errorf("failed to load accounts: %w", err)
	}

	for _, info := range infos {
		switch decideRetention(info, now) {
		case backfillActivity:
			if err := s.db.RecordUserActivity(info.UserID, now); err != nil {
				return result, fmt.Errorf("failed to backfill activity for user %d: %w", info.UserID, err)
			}
			result.AccountsBackfilled++
		case deleteAccount:
			log.Printf("Retention: deleting user %d, inactive since %s", info.UserID, info.LastSeenAt.Format(time.RFC3339))
			if err := s.userDataService.DeleteAccount(info.UserID); err != nil {
				log.Printf("Retention: failed to delete user %d: %v", info.UserID, err)
				result.DeletionsFailed++
				continue
			}
			result.AccountsDeleted++
		}
	}

	cleared, err := s.db.ClearAuditLogIPsBefore(now.AddDate(0, -auditLogIPRetentionMonths, 0))
	result.AuditIPsCleared = cleared
	if err != nil {
		return result, fmt.Errorf("failed to expire audit log IPs: %w", err)
	}

	if result.DeletionsFailed > 0 {
		return result, fmt.Errorf("%d account deletions failed", result.DeletionsFailed)
	}
	return result, nil
}
