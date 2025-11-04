package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"goread2/internal/database"
	"goread2/internal/services"
)

// Mock database for admin handler testing
type mockDBAdminHandler struct {
	users               map[int]*database.User
	auditLogs           []database.AuditLog
	shouldFailGetUser   bool
	shouldFailSetAdmin  bool
	shouldFailGrantFree bool
	shouldFailAudit     bool
	auditCallCount      int
}

func newMockDBAdminHandler() *mockDBAdminHandler {
	return &mockDBAdminHandler{
		users:     make(map[int]*database.User),
		auditLogs: make([]database.AuditLog, 0),
	}
}

func (m *mockDBAdminHandler) GetUserByEmail(email string) (*database.User, error) {
	if m.shouldFailGetUser {
		return nil, errors.New("database error")
	}
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockDBAdminHandler) SetUserAdmin(userID int, isAdmin bool) error {
	if m.shouldFailSetAdmin {
		return errors.New("failed to set admin")
	}
	if user, exists := m.users[userID]; exists {
		user.IsAdmin = isAdmin
	}
	return nil
}

func (m *mockDBAdminHandler) GrantFreeMonths(userID int, months int) error {
	if m.shouldFailGrantFree {
		return errors.New("failed to grant free months")
	}
	if user, exists := m.users[userID]; exists {
		user.FreeMonthsRemaining += months
	}
	return nil
}

func (m *mockDBAdminHandler) CreateAuditLog(log *database.AuditLog) error {
	m.auditCallCount++
	if m.shouldFailAudit {
		return errors.New("failed to create audit log")
	}
	log.ID = len(m.auditLogs) + 1
	m.auditLogs = append(m.auditLogs, *log)
	return nil
}

func (m *mockDBAdminHandler) GetAuditLogs(limit, offset int, filters map[string]interface{}) ([]database.AuditLog, error) {
	return m.auditLogs, nil
}

// Stub methods to satisfy interface
func (m *mockDBAdminHandler) Close() error                                              { return nil }
func (m *mockDBAdminHandler) CreateUser(*database.User) error                           { return nil }
func (m *mockDBAdminHandler) GetUserByGoogleID(string) (*database.User, error)          { return nil, nil }
func (m *mockDBAdminHandler) GetUserByID(int) (*database.User, error)                   { return nil, nil }
func (m *mockDBAdminHandler) UpdateUserSubscription(int, string, string, time.Time, time.Time) error { return nil }
func (m *mockDBAdminHandler) IsUserSubscriptionActive(int) (bool, error)                { return false, nil }
func (m *mockDBAdminHandler) GetUserFeedCount(int) (int, error)                         { return 0, nil }
func (m *mockDBAdminHandler) AddFeed(*database.Feed) error                              { return nil }
func (m *mockDBAdminHandler) UpdateFeed(*database.Feed) error                           { return nil }
func (m *mockDBAdminHandler) UpdateFeedTracking(int, time.Time, time.Time, int) error   { return nil }
func (m *mockDBAdminHandler) GetFeeds() ([]database.Feed, error)                        { return nil, nil }
func (m *mockDBAdminHandler) GetFeedByURL(string) (*database.Feed, error)               { return nil, nil }
func (m *mockDBAdminHandler) GetUserFeeds(int) ([]database.Feed, error)                 { return nil, nil }
func (m *mockDBAdminHandler) GetAllUserFeeds() ([]database.Feed, error)                 { return nil, nil }
func (m *mockDBAdminHandler) DeleteFeed(int) error                                      { return nil }
func (m *mockDBAdminHandler) SubscribeUserToFeed(int, int) error                        { return nil }
func (m *mockDBAdminHandler) UnsubscribeUserFromFeed(int, int) error                    { return nil }
func (m *mockDBAdminHandler) AddArticle(*database.Article) error                        { return nil }
func (m *mockDBAdminHandler) GetArticles(int) ([]database.Article, error)               { return nil, nil }
func (m *mockDBAdminHandler) FindArticleByURL(string) (*database.Article, error)        { return nil, nil }
func (m *mockDBAdminHandler) GetUserArticles(int) ([]database.Article, error)           { return nil, nil }
func (m *mockDBAdminHandler) GetUserArticlesPaginated(int, int, string, bool) (*database.ArticlePaginationResult, error) { return &database.ArticlePaginationResult{}, nil }
func (m *mockDBAdminHandler) GetUserFeedArticles(int, int) ([]database.Article, error)  { return nil, nil }
func (m *mockDBAdminHandler) GetUserArticleStatus(int, int) (*database.UserArticle, error) { return nil, nil }
func (m *mockDBAdminHandler) SetUserArticleStatus(int, int, bool, bool) error           { return nil }
func (m *mockDBAdminHandler) BatchSetUserArticleStatus(int, []database.Article, bool, bool) error { return nil }
func (m *mockDBAdminHandler) MarkUserArticleRead(int, int, bool) error                  { return nil }
func (m *mockDBAdminHandler) ToggleUserArticleStar(int, int) error                      { return nil }
func (m *mockDBAdminHandler) GetUserUnreadCounts(int) (map[int]int, error)              { return nil, nil }
func (m *mockDBAdminHandler) UpdateFeedLastFetch(int, time.Time) error                  { return nil }
func (m *mockDBAdminHandler) UpdateUserMaxArticlesOnFeedAdd(int, int) error             { return nil }
func (m *mockDBAdminHandler) CreateSession(*database.Session) error                     { return nil }
func (m *mockDBAdminHandler) GetSession(string) (*database.Session, error)              { return nil, nil }
func (m *mockDBAdminHandler) DeleteSession(string) error                                { return nil }
func (m *mockDBAdminHandler) DeleteExpiredSessions() error                              { return nil }

func TestNewAdminHandler(t *testing.T) {
	// Create mock services
	mockSubscriptionService := &services.SubscriptionService{}
	mockAuditService := &services.AuditService{}

	handler := NewAdminHandler(mockSubscriptionService, mockAuditService)

	if handler == nil {
		t.Fatal("NewAdminHandler returned nil")
	}

	if handler.subscriptionService != mockSubscriptionService {
		t.Error("AdminHandler subscription service not set correctly")
	}

	if handler.auditService != mockAuditService {
		t.Error("AdminHandler audit service not set correctly")
	}
}

func TestGetAuditLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newMockDBAdminHandler()
	subscriptionService := services.NewSubscriptionService(db)
	auditService := services.NewAuditService(db)
	handler := NewAdminHandler(subscriptionService, auditService)

	// Create test audit logs
	testLogs := []database.AuditLog{
		{
			ID:               1,
			Timestamp:        time.Now(),
			AdminUserID:      1,
			AdminEmail:       "admin@example.com",
			OperationType:    "grant_admin",
			TargetUserID:     2,
			TargetUserEmail:  "user@example.com",
			OperationDetails: `{"is_admin":true}`,
			IPAddress:        "192.168.1.1",
			Result:           "success",
		},
		{
			ID:               2,
			Timestamp:        time.Now().Add(-1 * time.Hour),
			AdminUserID:      1,
			AdminEmail:       "admin@example.com",
			OperationType:    "grant_free_months",
			TargetUserID:     3,
			TargetUserEmail:  "user2@example.com",
			OperationDetails: `{"months_granted":6}`,
			IPAddress:        "192.168.1.1",
			Result:           "success",
		},
	}
	db.auditLogs = testLogs

	t.Run("get all logs with default parameters", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/admin/audit-logs", nil)

		handler.GetAuditLogs(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		logs, ok := response["logs"].([]interface{})
		if !ok {
			t.Fatal("Response should contain logs array")
		}

		if len(logs) != 2 {
			t.Errorf("Expected 2 logs, got %d", len(logs))
		}
	})

	t.Run("get logs with limit parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/admin/audit-logs?limit=1", nil)

		handler.GetAuditLogs(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["limit"] != float64(1) {
			t.Errorf("Expected limit 1, got %v", response["limit"])
		}
	})

	t.Run("get logs with offset parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/admin/audit-logs?offset=1", nil)

		handler.GetAuditLogs(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["offset"] != float64(1) {
			t.Errorf("Expected offset 1, got %v", response["offset"])
		}
	})
}

func TestSetAdminStatusAuditLogging(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newMockDBAdminHandler()
	subscriptionService := services.NewSubscriptionService(db)
	auditService := services.NewAuditService(db)
	handler := NewAdminHandler(subscriptionService, auditService)

	// Create admin user
	adminUser := &database.User{
		ID:      1,
		Email:   "admin@example.com",
		Name:    "Admin User",
		IsAdmin: true,
	}
	db.users[1] = adminUser

	// Create target user
	targetUser := &database.User{
		ID:      2,
		Email:   "user@example.com",
		Name:    "Target User",
		IsAdmin: false,
	}
	db.users[2] = targetUser

	t.Run("successful admin grant logs audit entry", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		requestBody := map[string]bool{"is_admin": true}
		bodyBytes, _ := json.Marshal(requestBody)
		c.Request = httptest.NewRequest("POST", "/admin/users/user@example.com/admin", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "email", Value: "user@example.com"}}

		// Set user in context
		c.Set("user", adminUser)

		handler.SetAdminStatus(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify audit log was created
		if db.auditCallCount != 1 {
			t.Errorf("Expected 1 audit log call, got %d", db.auditCallCount)
		}

		if len(db.auditLogs) != 1 {
			t.Fatalf("Expected 1 audit log, got %d", len(db.auditLogs))
		}

		log := db.auditLogs[0]
		if log.OperationType != "grant_admin" {
			t.Errorf("Expected operation_type 'grant_admin', got '%s'", log.OperationType)
		}
		if log.Result != "success" {
			t.Errorf("Expected result 'success', got '%s'", log.Result)
		}
		if log.AdminUserID != 1 {
			t.Errorf("Expected admin_user_id 1, got %d", log.AdminUserID)
		}
		if log.TargetUserID != 2 {
			t.Errorf("Expected target_user_id 2, got %d", log.TargetUserID)
		}
	})
}

func TestGrantFreeMonthsAuditLogging(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newMockDBAdminHandler()
	subscriptionService := services.NewSubscriptionService(db)
	auditService := services.NewAuditService(db)
	handler := NewAdminHandler(subscriptionService, auditService)

	// Create admin user
	adminUser := &database.User{
		ID:      1,
		Email:   "admin@example.com",
		Name:    "Admin User",
		IsAdmin: true,
	}
	db.users[1] = adminUser

	// Create target user
	targetUser := &database.User{
		ID:                  2,
		Email:               "user@example.com",
		Name:                "Target User",
		FreeMonthsRemaining: 0,
	}
	db.users[2] = targetUser

	t.Run("successful free months grant logs audit entry", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		requestBody := map[string]int{"months": 6}
		bodyBytes, _ := json.Marshal(requestBody)
		c.Request = httptest.NewRequest("POST", "/admin/users/user@example.com/free-months", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{gin.Param{Key: "email", Value: "user@example.com"}}

		// Set user in context
		c.Set("user", adminUser)

		handler.GrantFreeMonths(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify audit log was created
		if db.auditCallCount != 1 {
			t.Errorf("Expected 1 audit log call, got %d", db.auditCallCount)
		}

		if len(db.auditLogs) != 1 {
			t.Fatalf("Expected 1 audit log, got %d", len(db.auditLogs))
		}

		log := db.auditLogs[0]
		if log.OperationType != "grant_free_months" {
			t.Errorf("Expected operation_type 'grant_free_months', got '%s'", log.OperationType)
		}
		if log.Result != "success" {
			t.Errorf("Expected result 'success', got '%s'", log.Result)
		}
	})
}
