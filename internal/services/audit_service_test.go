package services

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"goread2/internal/database"
)

// mockDBAudit implements database.Database interface for audit testing
type mockDBAudit struct {
	auditLogs         []database.AuditLog
	shouldFailCreate  bool
	shouldFailGet     bool
	createCallCount   int
	getCallCount      int
	lastGetFilters    map[string]interface{}
	lastGetLimit      int
	lastGetOffset     int
}

func newMockDBAudit() *mockDBAudit {
	return &mockDBAudit{
		auditLogs: make([]database.AuditLog, 0),
	}
}

func (m *mockDBAudit) CreateAuditLog(log *database.AuditLog) error {
	m.createCallCount++
	if m.shouldFailCreate {
		return errors.New("database error creating audit log")
	}
	log.ID = len(m.auditLogs) + 1
	log.Timestamp = time.Now()
	m.auditLogs = append(m.auditLogs, *log)
	return nil
}

func (m *mockDBAudit) GetAuditLogs(limit, offset int, filters map[string]interface{}) ([]database.AuditLog, error) {
	m.getCallCount++
	m.lastGetLimit = limit
	m.lastGetOffset = offset
	m.lastGetFilters = filters

	if m.shouldFailGet {
		return nil, errors.New("database error getting audit logs")
	}

	// Apply filters
	filtered := make([]database.AuditLog, 0)
	for _, log := range m.auditLogs {
		match := true

		if adminID, ok := filters["admin_user_id"].(int); ok {
			if log.AdminUserID != adminID {
				match = false
			}
		}

		if targetID, ok := filters["target_user_id"].(int); ok {
			if log.TargetUserID != targetID {
				match = false
			}
		}

		if opType, ok := filters["operation_type"].(string); ok {
			if log.OperationType != opType {
				match = false
			}
		}

		if match {
			filtered = append(filtered, log)
		}
	}

	// Apply pagination
	start := offset
	if start > len(filtered) {
		return []database.AuditLog{}, nil
	}

	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

// Stub methods to satisfy interface
func (m *mockDBAudit) Close() error                                              { return nil }
func (m *mockDBAudit) CreateUser(*database.User) error                           { return nil }
func (m *mockDBAudit) GetUserByGoogleID(string) (*database.User, error)          { return nil, nil }
func (m *mockDBAudit) GetUserByID(int) (*database.User, error)                   { return nil, nil }
func (m *mockDBAudit) GetUserByEmail(string) (*database.User, error)             { return nil, nil }
func (m *mockDBAudit) UpdateUserSubscription(int, string, string, time.Time, time.Time) error { return nil }
func (m *mockDBAudit) IsUserSubscriptionActive(int) (bool, error)                { return false, nil }
func (m *mockDBAudit) GetUserFeedCount(int) (int, error)                         { return 0, nil }
func (m *mockDBAudit) SetUserAdmin(int, bool) error                              { return nil }
func (m *mockDBAudit) GrantFreeMonths(int, int) error                            { return nil }
func (m *mockDBAudit) AddFeed(*database.Feed) error                              { return nil }
func (m *mockDBAudit) UpdateFeed(*database.Feed) error                           { return nil }
func (m *mockDBAudit) GetFeeds() ([]database.Feed, error)                        { return nil, nil }
func (m *mockDBAudit) GetFeedByURL(string) (*database.Feed, error)               { return nil, nil }
func (m *mockDBAudit) GetUserFeeds(int) ([]database.Feed, error)                 { return nil, nil }
func (m *mockDBAudit) GetAllUserFeeds() ([]database.Feed, error)                 { return nil, nil }
func (m *mockDBAudit) DeleteFeed(int) error                                      { return nil }
func (m *mockDBAudit) SubscribeUserToFeed(int, int) error                        { return nil }
func (m *mockDBAudit) UnsubscribeUserFromFeed(int, int) error                    { return nil }
func (m *mockDBAudit) AddArticle(*database.Article) error                        { return nil }
func (m *mockDBAudit) GetArticles(int) ([]database.Article, error)               { return nil, nil }
func (m *mockDBAudit) FindArticleByURL(string) (*database.Article, error)        { return nil, nil }
func (m *mockDBAudit) GetUserArticles(int) ([]database.Article, error)           { return nil, nil }
func (m *mockDBAudit) GetUserArticlesPaginated(int, int, int, bool) ([]database.Article, error) { return nil, nil }
func (m *mockDBAudit) GetUserFeedArticles(int, int) ([]database.Article, error)  { return nil, nil }
func (m *mockDBAudit) GetUserArticleStatus(int, int) (*database.UserArticle, error) { return nil, nil }
func (m *mockDBAudit) SetUserArticleStatus(int, int, bool, bool) error           { return nil }
func (m *mockDBAudit) BatchSetUserArticleStatus(int, []database.Article, bool, bool) error { return nil }
func (m *mockDBAudit) MarkUserArticleRead(int, int, bool) error                  { return nil }
func (m *mockDBAudit) ToggleUserArticleStar(int, int) error                      { return nil }
func (m *mockDBAudit) GetUserUnreadCounts(int) (map[int]int, error)              { return nil, nil }
func (m *mockDBAudit) GetAllArticles() ([]database.Article, error)               { return nil, nil }
func (m *mockDBAudit) UpdateFeedLastFetch(int, time.Time) error                  { return nil }
func (m *mockDBAudit) UpdateUserMaxArticlesOnFeedAdd(int, int) error             { return nil }
func (m *mockDBAudit) CreateSession(*database.Session) error                     { return nil }
func (m *mockDBAudit) GetSession(string) (*database.Session, error)              { return nil, nil }
func (m *mockDBAudit) DeleteSession(string) error                                { return nil }
func (m *mockDBAudit) DeleteExpiredSessions() error                              { return nil }

func TestNewAuditService(t *testing.T) {
	db := newMockDBAudit()
	service := NewAuditService(db)

	if service == nil {
		t.Fatal("NewAuditService returned nil")
	}

	if service.db == nil {
		t.Error("AuditService database not set")
	}
}

func TestLogSuccess(t *testing.T) {
	db := newMockDBAudit()
	service := NewAuditService(db)

	details := map[string]interface{}{
		"is_admin":  true,
		"user_name": "Test User",
	}

	err := service.LogSuccess(1, "admin@example.com", "grant_admin", 2, "user@example.com", details, "192.168.1.1")

	if err != nil {
		t.Fatalf("LogSuccess failed: %v", err)
	}

	if db.createCallCount != 1 {
		t.Errorf("CreateAuditLog called %d times, want 1", db.createCallCount)
	}

	if len(db.auditLogs) != 1 {
		t.Fatalf("Expected 1 audit log, got %d", len(db.auditLogs))
	}

	log := db.auditLogs[0]
	if log.AdminUserID != 1 {
		t.Errorf("AdminUserID = %d, want 1", log.AdminUserID)
	}
	if log.AdminEmail != "admin@example.com" {
		t.Errorf("AdminEmail = %s, want admin@example.com", log.AdminEmail)
	}
	if log.OperationType != "grant_admin" {
		t.Errorf("OperationType = %s, want grant_admin", log.OperationType)
	}
	if log.TargetUserID != 2 {
		t.Errorf("TargetUserID = %d, want 2", log.TargetUserID)
	}
	if log.TargetUserEmail != "user@example.com" {
		t.Errorf("TargetUserEmail = %s, want user@example.com", log.TargetUserEmail)
	}
	if log.IPAddress != "192.168.1.1" {
		t.Errorf("IPAddress = %s, want 192.168.1.1", log.IPAddress)
	}
	if log.Result != "success" {
		t.Errorf("Result = %s, want success", log.Result)
	}
	if log.ErrorMessage != "" {
		t.Errorf("ErrorMessage = %s, want empty", log.ErrorMessage)
	}

	// Verify details JSON
	var parsedDetails map[string]interface{}
	if err := json.Unmarshal([]byte(log.OperationDetails), &parsedDetails); err != nil {
		t.Fatalf("Failed to parse operation details: %v", err)
	}
	if parsedDetails["is_admin"] != true {
		t.Error("Details is_admin should be true")
	}
	if parsedDetails["user_name"] != "Test User" {
		t.Error("Details user_name should be Test User")
	}
}

func TestLogFailure(t *testing.T) {
	db := newMockDBAudit()
	service := NewAuditService(db)

	details := map[string]interface{}{
		"months_granted": 6,
	}

	err := service.LogFailure(1, "admin@example.com", "grant_free_months", 2, "user@example.com", details, "192.168.1.1", "user not found")

	if err != nil {
		t.Fatalf("LogFailure failed: %v", err)
	}

	if len(db.auditLogs) != 1 {
		t.Fatalf("Expected 1 audit log, got %d", len(db.auditLogs))
	}

	log := db.auditLogs[0]
	if log.Result != "failure" {
		t.Errorf("Result = %s, want failure", log.Result)
	}
	if log.ErrorMessage != "user not found" {
		t.Errorf("ErrorMessage = %s, want 'user not found'", log.ErrorMessage)
	}
}

func TestLogSuccessWithEmptyDetails(t *testing.T) {
	db := newMockDBAudit()
	service := NewAuditService(db)

	err := service.LogSuccess(1, "admin@example.com", "view_user_info", 2, "user@example.com", nil, "192.168.1.1")

	if err != nil {
		t.Fatalf("LogSuccess with nil details failed: %v", err)
	}

	if len(db.auditLogs) != 1 {
		t.Fatalf("Expected 1 audit log, got %d", len(db.auditLogs))
	}

	log := db.auditLogs[0]
	// When details is nil, OperationDetails should be empty string
	if log.OperationDetails != "" {
		t.Errorf("OperationDetails = %s, want empty string", log.OperationDetails)
	}
}

func TestLogSuccessWithDatabaseError(t *testing.T) {
	db := newMockDBAudit()
	db.shouldFailCreate = true
	service := NewAuditService(db)

	err := service.LogSuccess(1, "admin@example.com", "grant_admin", 2, "user@example.com", nil, "192.168.1.1")

	if err == nil {
		t.Error("Expected error when database fails, got nil")
	}
}

func TestGetAuditLogs(t *testing.T) {
	db := newMockDBAudit()
	service := NewAuditService(db)

	// Create test audit logs
	testLogs := []database.AuditLog{
		{
			ID:               1,
			AdminUserID:      1,
			AdminEmail:       "admin@example.com",
			OperationType:    "grant_admin",
			TargetUserID:     2,
			TargetUserEmail:  "user1@example.com",
			OperationDetails: "{}",
			IPAddress:        "192.168.1.1",
			Result:           "success",
		},
		{
			ID:               2,
			AdminUserID:      1,
			AdminEmail:       "admin@example.com",
			OperationType:    "grant_free_months",
			TargetUserID:     3,
			TargetUserEmail:  "user2@example.com",
			OperationDetails: "{}",
			IPAddress:        "192.168.1.1",
			Result:           "success",
		},
		{
			ID:               3,
			AdminUserID:      2,
			AdminEmail:       "admin2@example.com",
			OperationType:    "grant_admin",
			TargetUserID:     4,
			TargetUserEmail:  "user3@example.com",
			OperationDetails: "{}",
			IPAddress:        "192.168.1.2",
			Result:           "failure",
			ErrorMessage:     "user not found",
		},
	}
	db.auditLogs = testLogs

	t.Run("get all logs with default pagination", func(t *testing.T) {
		logs, err := service.GetAuditLogs(50, 0, nil)
		if err != nil {
			t.Fatalf("GetAuditLogs failed: %v", err)
		}
		if len(logs) != 3 {
			t.Errorf("Expected 3 logs, got %d", len(logs))
		}
		if db.lastGetLimit != 50 {
			t.Errorf("Expected limit 50, got %d", db.lastGetLimit)
		}
		if db.lastGetOffset != 0 {
			t.Errorf("Expected offset 0, got %d", db.lastGetOffset)
		}
	})

	t.Run("get logs with pagination", func(t *testing.T) {
		logs, err := service.GetAuditLogs(2, 1, nil)
		if err != nil {
			t.Fatalf("GetAuditLogs failed: %v", err)
		}
		if len(logs) != 2 {
			t.Errorf("Expected 2 logs with limit=2, offset=1, got %d", len(logs))
		}
	})

	t.Run("filter by admin_user_id", func(t *testing.T) {
		filters := map[string]interface{}{
			"admin_user_id": 1,
		}
		logs, err := service.GetAuditLogs(50, 0, filters)
		if err != nil {
			t.Fatalf("GetAuditLogs failed: %v", err)
		}
		if len(logs) != 2 {
			t.Errorf("Expected 2 logs for admin_user_id=1, got %d", len(logs))
		}
		for _, log := range logs {
			if log.AdminUserID != 1 {
				t.Errorf("Expected AdminUserID=1, got %d", log.AdminUserID)
			}
		}
	})

	t.Run("filter by target_user_id", func(t *testing.T) {
		filters := map[string]interface{}{
			"target_user_id": 2,
		}
		logs, err := service.GetAuditLogs(50, 0, filters)
		if err != nil {
			t.Fatalf("GetAuditLogs failed: %v", err)
		}
		if len(logs) != 1 {
			t.Errorf("Expected 1 log for target_user_id=2, got %d", len(logs))
		}
		if logs[0].TargetUserID != 2 {
			t.Errorf("Expected TargetUserID=2, got %d", logs[0].TargetUserID)
		}
	})

	t.Run("filter by operation_type", func(t *testing.T) {
		filters := map[string]interface{}{
			"operation_type": "grant_admin",
		}
		logs, err := service.GetAuditLogs(50, 0, filters)
		if err != nil {
			t.Fatalf("GetAuditLogs failed: %v", err)
		}
		if len(logs) != 2 {
			t.Errorf("Expected 2 logs for operation_type=grant_admin, got %d", len(logs))
		}
		for _, log := range logs {
			if log.OperationType != "grant_admin" {
				t.Errorf("Expected OperationType=grant_admin, got %s", log.OperationType)
			}
		}
	})

	t.Run("filter with multiple criteria", func(t *testing.T) {
		filters := map[string]interface{}{
			"admin_user_id":  1,
			"operation_type": "grant_admin",
		}
		logs, err := service.GetAuditLogs(50, 0, filters)
		if err != nil {
			t.Fatalf("GetAuditLogs failed: %v", err)
		}
		if len(logs) != 1 {
			t.Errorf("Expected 1 log with multiple filters, got %d", len(logs))
		}
		if logs[0].AdminUserID != 1 || logs[0].OperationType != "grant_admin" {
			t.Error("Log doesn't match filter criteria")
		}
	})

	t.Run("no results for non-matching filter", func(t *testing.T) {
		filters := map[string]interface{}{
			"admin_user_id": 999,
		}
		logs, err := service.GetAuditLogs(50, 0, filters)
		if err != nil {
			t.Fatalf("GetAuditLogs failed: %v", err)
		}
		if len(logs) != 0 {
			t.Errorf("Expected 0 logs for non-matching filter, got %d", len(logs))
		}
	})
}

func TestGetAuditLogsWithDatabaseError(t *testing.T) {
	db := newMockDBAudit()
	db.shouldFailGet = true
	service := NewAuditService(db)

	_, err := service.GetAuditLogs(50, 0, nil)
	if err == nil {
		t.Error("Expected error when database fails, got nil")
	}
}

func TestCLIAuditLogging(t *testing.T) {
	db := newMockDBAudit()
	service := NewAuditService(db)

	// Simulate CLI operation
	err := service.LogSuccess(0, "CLI_ADMIN", "grant_admin", 5, "user@example.com", nil, "CLI")

	if err != nil {
		t.Fatalf("CLI audit logging failed: %v", err)
	}

	if len(db.auditLogs) != 1 {
		t.Fatalf("Expected 1 audit log, got %d", len(db.auditLogs))
	}

	log := db.auditLogs[0]
	if log.AdminUserID != 0 {
		t.Errorf("CLI AdminUserID should be 0, got %d", log.AdminUserID)
	}
	if log.AdminEmail != "CLI_ADMIN" {
		t.Errorf("CLI AdminEmail should be CLI_ADMIN, got %s", log.AdminEmail)
	}
	if log.IPAddress != "CLI" {
		t.Errorf("CLI IPAddress should be CLI, got %s", log.IPAddress)
	}
}
