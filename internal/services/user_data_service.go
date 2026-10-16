package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/secrets"
)

const (
	// exportArticlePageSize bounds memory per page of read/starred history.
	// Datastore loads full Article entities (including content) to join titles
	// and URLs, so pages stay small.
	exportArticlePageSize = 200
	exportAuditPageSize   = 500
)

// UserDataService serves GDPR requests: it assembles a user's personal data
// for access and portability, and deletes it for erasure.
type UserDataService struct {
	db              database.Database
	customerDeleter CustomerDeleter
}

// CustomerDeleter removes a user's customer records from the payment
// provider. *PaymentService satisfies it.
type CustomerDeleter interface {
	DeleteCustomerForUser(user *database.User) error
}

func NewUserDataService(db database.Database) *UserDataService {
	return &UserDataService{db: db}
}

// SetCustomerDeleter enables Stripe customer deletion during account
// deletion. Called from main only when subscriptions are enabled.
func (s *UserDataService) SetCustomerDeleter(cd CustomerDeleter) {
	s.customerDeleter = cd
}

// DeleteAccount permanently deletes userID's account. While the user record
// still exists it deletes the user's Stripe customer (cancelling any
// subscription immediately) and writes a tombstone so that signing in again
// does not start a new free trial; it then deletes all of the user's data,
// removing the user record last. Each step is idempotent, so a failed run can
// be retried, and running it again after the user is gone sweeps up any rows
// recreated in the meantime by requests on stale cached sessions.
func (s *UserDataService) DeleteAccount(userID int) error {
	user, err := s.db.GetUserByID(userID)
	if err != nil && !errors.Is(err, database.ErrUserNotFound) {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user != nil {
		if s.customerDeleter != nil {
			if err := s.customerDeleter.DeleteCustomerForUser(user); err != nil {
				return fmt.Errorf("failed to delete Stripe customer: %w", err)
			}
		}
		if err := s.writeTombstone(user); err != nil {
			return err
		}
	}

	if err := s.db.DeleteUserAccount(userID); err != nil {
		return fmt.Errorf("failed to delete account data: %w", err)
	}
	return nil
}

func (s *UserDataService) writeTombstone(user *database.User) error {
	id, err := secrets.AccountTombstoneID(context.Background(), user.GoogleID)
	if id == "" {
		// Deletion must not depend on the anti-abuse tombstone, so a missing
		// key only costs the re-signup trial check.
		log.Printf("Warning: account tombstone key unavailable, deleting user %d without a tombstone: %v", user.ID, err)
		return nil
	}
	if err := s.db.CreateAccountTombstone(id); err != nil {
		return fmt.Errorf("failed to write account tombstone: %w", err)
	}
	return nil
}

// UserDataExport holds the parts of a data export that are small enough to
// load up front. Read/starred history can be large, so WriteExport streams it
// after these fields.
type UserDataExport struct {
	ExportedAt time.Time            `json:"exported_at"`
	Account    *database.User       `json:"account"`
	LastSeenAt time.Time            `json:"last_seen_at,omitzero"`
	Feeds      []ExportedFeed       `json:"feeds"`
	Sessions   []ExportedSession    `json:"sessions"`
	AuditLog   []ExportedAuditEntry `json:"audit_log"`
}

type ExportedFeed struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ExportedSession omits the session ID, which is a live credential.
type ExportedSession struct {
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ExportedAuditEntry is an admin action that involved the user, either as the
// target ("subject") or as the admin who performed it ("actor"). It omits the
// other party's email and IP address, which are not the user's data.
type ExportedAuditEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	Role          string    `json:"role"`
	OperationType string    `json:"operation_type"`
	Details       string    `json:"details,omitzero"`
	Result        string    `json:"result"`
}

// PrepareExport loads everything in a data export except read/starred
// history. Callers should call it before sending response headers, so that a
// failure here can still be reported as an error status.
func (s *UserDataService) PrepareExport(userID int) (*UserDataExport, error) {
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	feeds, err := s.db.GetUserFeeds(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feeds: %w", err)
	}
	exportedFeeds := make([]ExportedFeed, len(feeds))
	for i, feed := range feeds {
		exportedFeeds[i] = ExportedFeed{ID: feed.ID, Title: feed.Title, URL: feed.URL}
	}

	lastSeen, err := s.db.GetUserLastSeen(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last activity: %w", err)
	}

	sessions, err := s.db.GetUserSessions(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}
	exportedSessions := make([]ExportedSession, len(sessions))
	for i, session := range sessions {
		exportedSessions[i] = ExportedSession{CreatedAt: session.CreatedAt, ExpiresAt: session.ExpiresAt}
	}

	auditLog, err := s.exportAuditLog(userID)
	if err != nil {
		return nil, err
	}

	return &UserDataExport{
		ExportedAt: time.Now().UTC(),
		Account:    user,
		LastSeenAt: lastSeen,
		Feeds:      exportedFeeds,
		Sessions:   exportedSessions,
		AuditLog:   auditLog,
	}, nil
}

// exportAuditLog collects audit entries where userID is the target or the
// admin, newest first. An admin acting on their own account appears once.
func (s *UserDataService) exportAuditLog(userID int) ([]ExportedAuditEntry, error) {
	seen := make(map[int]bool)
	entries := []ExportedAuditEntry{}

	for _, query := range []struct {
		filter string
		role   string
	}{
		{"target_user_id", "subject"},
		{"admin_user_id", "actor"},
	} {
		for offset := 0; ; offset += exportAuditPageSize {
			logs, err := s.db.GetAuditLogs(exportAuditPageSize, offset, map[string]interface{}{query.filter: userID})
			if err != nil {
				return nil, fmt.Errorf("failed to get audit logs: %w", err)
			}
			for _, log := range logs {
				if seen[log.ID] {
					continue
				}
				seen[log.ID] = true
				entries = append(entries, ExportedAuditEntry{
					Timestamp:     log.Timestamp,
					Role:          query.role,
					OperationType: log.OperationType,
					Details:       log.OperationDetails,
					Result:        log.Result,
				})
			}
			if len(logs) < exportAuditPageSize {
				break
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Timestamp.After(entries[j].Timestamp) })
	return entries, nil
}

// WriteExport writes export as a single JSON object, followed by the user's
// read/starred history under "articles", fetched and written one page at a
// time. If fetching a page fails partway through, the output is left
// truncated, which makes it invalid JSON rather than a silently incomplete
// export.
func (s *UserDataService) WriteExport(w io.Writer, userID int, export *UserDataExport) error {
	head, err := json.Marshal(export)
	if err != nil {
		return fmt.Errorf("failed to encode export: %w", err)
	}
	// Reopen the object to append the streamed articles array.
	if _, err := w.Write(head[:len(head)-1]); err != nil {
		return err
	}
	if _, err := io.WriteString(w, `,"articles":[`); err != nil {
		return err
	}

	first := true
	cursor := ""
	for {
		records, next, err := s.db.GetUserArticleHistoryPage(userID, cursor, exportArticlePageSize)
		if err != nil {
			return fmt.Errorf("failed to get article history: %w", err)
		}
		for _, record := range records {
			encoded, err := json.Marshal(record)
			if err != nil {
				return fmt.Errorf("failed to encode article record: %w", err)
			}
			if !first {
				if _, err := io.WriteString(w, ","); err != nil {
					return err
				}
			}
			first = false
			if _, err := w.Write(encoded); err != nil {
				return err
			}
		}
		if next == "" {
			break
		}
		cursor = next
	}

	_, err = io.WriteString(w, "]}")
	return err
}
