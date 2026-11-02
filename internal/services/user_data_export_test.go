package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

// mockDBAuditLog serves GetAuditLogs from an in-memory log, applying the
// single equality filter and offset/limit paging the real implementations
// apply, and records each call. Any other Database call panics.
type mockDBAuditLog struct {
	database.Database
	logs  []database.AuditLog
	calls []string
}

func (m *mockDBAuditLog) GetAuditLogs(limit, offset int, filters map[string]interface{}) ([]database.AuditLog, error) {
	if len(filters) != 1 {
		return nil, fmt.Errorf("want exactly one filter, got %v", filters)
	}
	var matched []database.AuditLog
	for key, value := range filters {
		m.calls = append(m.calls, fmt.Sprintf("%s@%d", key, offset))
		for _, log := range m.logs {
			switch key {
			case "target_user_id":
				if log.TargetUserID == value {
					matched = append(matched, log)
				}
			case "admin_user_id":
				if log.AdminUserID == value {
					matched = append(matched, log)
				}
			default:
				return nil, fmt.Errorf("unexpected filter %q", key)
			}
		}
	}
	if offset >= len(matched) {
		return nil, nil
	}
	return matched[offset:min(offset+limit, len(matched))], nil
}

func TestExportAuditLog(t *testing.T) {
	const userID, otherID = 7, 99
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	// More than one page of entries targeting the user, stored oldest first
	// so that the export has to sort them.
	targetCount := exportAuditPageSize + 1
	var logs []database.AuditLog
	for i := range targetCount {
		logs = append(logs, database.AuditLog{
			ID:            i + 1,
			Timestamp:     base.Add(time.Duration(i) * time.Minute),
			AdminUserID:   otherID,
			TargetUserID:  userID,
			OperationType: "grant_months",
			Result:        "success",
		})
	}
	// The user, as an admin, acting on their own account: matched by both
	// filters, so it must be exported once.
	selfAction := database.AuditLog{
		ID:            1000,
		Timestamp:     base.Add(-time.Hour),
		AdminUserID:   userID,
		TargetUserID:  userID,
		OperationType: "set_admin",
		Result:        "success",
	}
	// The user, as an admin, acting on someone else.
	actorAction := database.AuditLog{
		ID:            1001,
		Timestamp:     base.Add(-2 * time.Hour),
		AdminUserID:   userID,
		TargetUserID:  otherID,
		OperationType: "remove_admin",
		Result:        "failure",
	}
	logs = append(logs, selfAction, actorAction)
	// An entry that involves only other users.
	logs = append(logs, database.AuditLog{ID: 1002, Timestamp: base, AdminUserID: otherID, TargetUserID: otherID + 1})

	db := &mockDBAuditLog{logs: logs}
	entries, err := NewUserDataService(db).exportAuditLog(userID)
	if err != nil {
		t.Fatalf("exportAuditLog() error = %v", err)
	}

	wantCalls := []string{"target_user_id@0", fmt.Sprintf("target_user_id@%d", exportAuditPageSize), "admin_user_id@0"}
	if strings.Join(db.calls, " ") != strings.Join(wantCalls, " ") {
		t.Errorf("GetAuditLogs calls = %v, want %v", db.calls, wantCalls)
	}

	if want := targetCount + 2; len(entries) != want {
		t.Fatalf("len(entries) = %d, want %d", len(entries), want)
	}

	for i := 1; i < len(entries); i++ {
		if entries[i].Timestamp.After(entries[i-1].Timestamp) {
			t.Fatalf("entries[%d] (%v) is newer than entries[%d] (%v), want newest first",
				i, entries[i].Timestamp, i-1, entries[i-1].Timestamp)
		}
	}
	if newest := base.Add(time.Duration(targetCount-1) * time.Minute); !entries[0].Timestamp.Equal(newest) {
		t.Errorf("entries[0].Timestamp = %v, want %v", entries[0].Timestamp, newest)
	}

	selfCount := 0
	for _, entry := range entries {
		if entry.OperationType == "set_admin" {
			selfCount++
			if entry.Role != "subject" {
				t.Errorf("self-action role = %q, want %q", entry.Role, "subject")
			}
		}
	}
	if selfCount != 1 {
		t.Errorf("self-action exported %d times, want 1", selfCount)
	}

	last := entries[len(entries)-1]
	want := ExportedAuditEntry{Timestamp: actorAction.Timestamp, Role: "actor", OperationType: "remove_admin", Result: "failure"}
	if last != want {
		t.Errorf("oldest entry = %+v, want %+v", last, want)
	}
}

func TestExportAuditLogError(t *testing.T) {
	dbErr := errors.New("datastore unavailable")
	_, err := NewUserDataService(&mockDBAuditLogErr{err: dbErr}).exportAuditLog(7)
	if !errors.Is(err, dbErr) {
		t.Fatalf("exportAuditLog() error = %v, want it to wrap %v", err, dbErr)
	}
}

type mockDBAuditLogErr struct {
	database.Database
	err error
}

func (m *mockDBAuditLogErr) GetAuditLogs(int, int, map[string]interface{}) ([]database.AuditLog, error) {
	return nil, m.err
}

// mockDBArticleHistory serves GetUserArticleHistoryPage from fixed pages. The
// cursor is the index of the next page. If failAt is non-negative, fetching
// that page fails.
type mockDBArticleHistory struct {
	database.Database
	pages  [][]database.UserArticleRecord
	failAt int
}

var errHistoryPage = errors.New("history page failed")

func (m *mockDBArticleHistory) GetUserArticleHistoryPage(userID int, cursor string, limit int) ([]database.UserArticleRecord, string, error) {
	page := 0
	if cursor != "" {
		var err error
		if page, err = strconv.Atoi(cursor); err != nil {
			return nil, "", fmt.Errorf("bad cursor %q", cursor)
		}
	}
	if page == m.failAt {
		return nil, "", errHistoryPage
	}
	if page >= len(m.pages) {
		return nil, "", nil
	}
	next := ""
	if page+1 < len(m.pages) {
		next = strconv.Itoa(page + 1)
	}
	return m.pages[page], next, nil
}

func TestWriteExport(t *testing.T) {
	export := &UserDataExport{
		ExportedAt: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC),
		Account:    &database.User{ID: 7, Email: "user@example.com"},
		Feeds:      []ExportedFeed{},
		Sessions:   []ExportedSession{},
		AuditLog:   []ExportedAuditEntry{},
	}
	record := func(id int) database.UserArticleRecord {
		return database.UserArticleRecord{ArticleID: id, FeedID: 1, Title: fmt.Sprintf("Article %d", id), IsRead: true}
	}

	tests := []struct {
		name         string
		pages        [][]database.UserArticleRecord
		failAt       int
		wantArticles []int
		wantErr      bool
	}{
		{
			name:         "no history ends with an empty articles array",
			pages:        nil,
			failAt:       -1,
			wantArticles: []int{},
		},
		{
			name:         "an empty first page ends with an empty articles array",
			pages:        [][]database.UserArticleRecord{{}},
			failAt:       -1,
			wantArticles: []int{},
		},
		{
			name:         "records from every page are written in order",
			pages:        [][]database.UserArticleRecord{{record(1), record(2)}, {record(3)}},
			failAt:       -1,
			wantArticles: []int{1, 2, 3},
		},
		{
			name:    "a failure on the first page leaves invalid JSON",
			pages:   [][]database.UserArticleRecord{{record(1)}},
			failAt:  0,
			wantErr: true,
		},
		{
			name:    "a failure partway through leaves invalid JSON",
			pages:   [][]database.UserArticleRecord{{record(1), record(2)}, {record(3)}},
			failAt:  1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &mockDBArticleHistory{pages: tt.pages, failAt: tt.failAt}
			var buf bytes.Buffer

			err := NewUserDataService(db).WriteExport(&buf, 7, export)

			if tt.wantErr {
				if !errors.Is(err, errHistoryPage) {
					t.Fatalf("WriteExport() error = %v, want it to wrap %v", err, errHistoryPage)
				}
				if json.Valid(buf.Bytes()) {
					t.Fatalf("output after a failed page is valid JSON, want it truncated: %s", buf.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("WriteExport() error = %v", err)
			}
			if !json.Valid(buf.Bytes()) {
				t.Fatalf("output is not valid JSON: %s", buf.String())
			}
			if len(tt.wantArticles) == 0 && !strings.HasSuffix(buf.String(), `"articles":[]}`) {
				t.Errorf("output ends %q, want suffix %q", buf.String()[max(0, buf.Len()-30):], `"articles":[]}`)
			}

			var decoded struct {
				Account  *database.User               `json:"account"`
				AuditLog []ExportedAuditEntry         `json:"audit_log"`
				Articles []database.UserArticleRecord `json:"articles"`
			}
			if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if decoded.Account == nil || decoded.Account.ID != 7 {
				t.Errorf("account = %+v, want ID 7", decoded.Account)
			}
			if decoded.Articles == nil {
				t.Fatalf("articles is missing or null, want an array")
			}
			gotIDs := []int{}
			for _, a := range decoded.Articles {
				gotIDs = append(gotIDs, a.ArticleID)
			}
			if fmt.Sprint(gotIDs) != fmt.Sprint(tt.wantArticles) {
				t.Errorf("article IDs = %v, want %v", gotIDs, tt.wantArticles)
			}
		})
	}
}
