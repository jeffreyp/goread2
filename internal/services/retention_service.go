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
	// auditLogIPSweepWindowMonths is how far before the IP retention cutoff
	// each run looks for entries to clear. Earlier runs have already cleared
	// anything older, so the window keeps each run's reads bounded as the
	// audit log grows. Three months still covers two consecutive missed
	// monthly runs.
	auditLogIPSweepWindowMonths = 3
	// pendingDeletionRetryDelay is how long a self-requested deletion
	// (DELETE /api/account) waits before Run retries it. It must comfortably
	// exceed both Cloud Tasks passes (the immediate attempt and the 15-minute
	// sweep), so Run only ever retries a deletion those have already given up
	// on, never one still in flight.
	pendingDeletionRetryDelay = 1 * time.Hour
)

// RetentionService applies GoRead2's data retention policy: it deletes
// inactive unpaid accounts, expires IP addresses in the audit log, and
// retries any self-requested deletion (DELETE /api/account) that Cloud Tasks
// gave up on. Run applies the whole policy monthly; RetryPendingDeletions
// applies only the last part, daily.
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
		if s.retryPendingDeletion(info, now, &result) {
			continue
		}

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

	ipCutoff := now.AddDate(0, -auditLogIPRetentionMonths, 0)
	cleared, err := s.db.ClearAuditLogIPsBetween(ipCutoff.AddDate(0, -auditLogIPSweepWindowMonths, 0), ipCutoff)
	result.AuditIPsCleared = cleared
	if err != nil {
		return result, fmt.Errorf("failed to expire audit log IPs: %w", err)
	}

	if result.DeletionsFailed > 0 {
		return result, fmt.Errorf("%d account deletions failed", result.DeletionsFailed)
	}
	return result, nil
}

// RetryPendingDeletions retries only the self-requested deletions
// (DELETE /api/account) that Cloud Tasks gave up on, without applying the
// rest of the retention policy. It runs daily, so a deletion that failed
// during an outage completes, and stops any Stripe billing, within a day
// rather than at the next monthly Run. Like Run, it continues past a failed
// deletion and then returns an error so that the task is retried.
func (s *RetentionService) RetryPendingDeletions(now time.Time) (RetentionResult, error) {
	var result RetentionResult

	infos, err := s.db.GetUserRetentionInfo()
	if err != nil {
		return result, fmt.Errorf("failed to load accounts: %w", err)
	}

	for _, info := range infos {
		s.retryPendingDeletion(info, now, &result)
	}

	if result.DeletionsFailed > 0 {
		return result, fmt.Errorf("%d account deletions failed", result.DeletionsFailed)
	}
	return result, nil
}

// retryPendingDeletion deletes the account if it has a self-requested
// deletion older than pendingDeletionRetryDelay, counting the outcome in
// result. It reports whether the account had such a deletion, in which case
// the caller must not apply any other retention rule to it.
func (s *RetentionService) retryPendingDeletion(info database.UserRetentionInfo, now time.Time, result *RetentionResult) bool {
	if info.DeletionRequestedAt.IsZero() || !info.DeletionRequestedAt.Before(now.Add(-pendingDeletionRetryDelay)) {
		return false
	}
	log.Printf("Retention: retrying pending deletion for user %d, requested at %s", info.UserID, info.DeletionRequestedAt.Format(time.RFC3339))
	if err := s.userDataService.DeleteAccount(info.UserID); err != nil {
		log.Printf("Pending account deletion retry failed for user %d: %v", info.UserID, err)
		result.DeletionsFailed++
		return true
	}
	result.AccountsDeleted++
	return true
}
