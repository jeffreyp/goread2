package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/database"
)

func TestNewMiddleware(t *testing.T) {
	db := newMockDB()
	sessionManager := NewSessionManager(db)

	middleware := NewMiddleware(sessionManager)

	if middleware == nil {
		t.Fatal("NewMiddleware returned nil")
	}

	if middleware.sessionManager != sessionManager {
		t.Error("Middleware session manager not set correctly")
	}
}

func TestGetUserFromContext(t *testing.T) {
	// Setup gin test mode
	gin.SetMode(gin.TestMode)

	user := &database.User{
		ID:    1,
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Test with user in context
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(string(UserContextKey), user)

	retrievedUser, exists := GetUserFromContext(c)
	if !exists {
		t.Error("GetUserFromContext should return true when user exists")
	}
	if retrievedUser != user {
		t.Error("GetUserFromContext returned wrong user")
	}

	// Test without user in context
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())

	retrievedUser2, exists2 := GetUserFromContext(c2)
	if exists2 {
		t.Error("GetUserFromContext should return false when user doesn't exist")
	}
	if retrievedUser2 != nil {
		t.Error("GetUserFromContext should return nil when user doesn't exist")
	}
}

func TestGetUserFromStdContext(t *testing.T) {
	user := &database.User{
		ID:    1,
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Test with user in context
	ctx := context.WithValue(context.Background(), UserContextKey, user)

	retrievedUser, exists := GetUserFromStdContext(ctx)
	if !exists {
		t.Error("GetUserFromStdContext should return true when user exists")
	}
	if retrievedUser != user {
		t.Error("GetUserFromStdContext returned wrong user")
	}

	// Test without user in context
	ctx2 := context.Background()

	retrievedUser2, exists2 := GetUserFromStdContext(ctx2)
	if exists2 {
		t.Error("GetUserFromStdContext should return false when user doesn't exist")
	}
	if retrievedUser2 != nil {
		t.Error("GetUserFromStdContext should return nil when user doesn't exist")
	}

	// Test with wrong type in context
	ctx3 := context.WithValue(context.Background(), UserContextKey, "not a user")

	retrievedUser3, exists3 := GetUserFromStdContext(ctx3)
	if exists3 {
		t.Error("GetUserFromStdContext should return false when value is wrong type")
	}
	if retrievedUser3 != nil {
		t.Error("GetUserFromStdContext should return nil when value is wrong type")
	}
}

func TestRequireAuth(t *testing.T) {
	// Setup gin test mode
	gin.SetMode(gin.TestMode)

	db := newMockDB()
	sessionManager := NewSessionManager(db)
	middleware := NewMiddleware(sessionManager)

	t.Run("no session returns JSON error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds", nil)

		middleware.RequireAuth()(c)

		if w.Code != 401 {
			t.Errorf("Expected status 401 (unauthorized), got %d", w.Code)
		}

		if !c.IsAborted() {
			t.Error("Expected request to be aborted")
		}
	})

	t.Run("valid session adds user to context", func(t *testing.T) {
		user := &database.User{
			ID:       1,
			Email:    "test@example.com",
			Name:     "Test User",
			GoogleID: "google-123",
		}

		// Create a valid session
		session, err := sessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/feeds", nil)
		c.Request.AddCookie(&http.Cookie{
			Name:  "session_id_local",
			Value: session.ID,
		})

		middleware.RequireAuth()(c)

		if c.IsAborted() {
			t.Error("Expected request not to be aborted")
		}

		retrievedUser, exists := GetUserFromContext(c)
		if !exists {
			t.Error("User not found in context")
		}
		if retrievedUser.ID != user.ID {
			t.Errorf("Wrong user in context: got %d, want %d", retrievedUser.ID, user.ID)
		}
	})
}

func TestRequireAuthPage(t *testing.T) {
	// Setup gin test mode
	gin.SetMode(gin.TestMode)

	db := newMockDB()
	sessionManager := NewSessionManager(db)
	middleware := NewMiddleware(sessionManager)

	// Test case: no session - should redirect to home page
	t.Run("no session redirects to home", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/account", nil)

		middleware.RequireAuthPage()(c)

		if w.Code != 302 {
			t.Errorf("Expected status 302 (redirect), got %d", w.Code)
		}

		location := w.Header().Get("Location")
		if location != "/" {
			t.Errorf("Expected redirect to '/', got '%s'", location)
		}

		if !c.IsAborted() {
			t.Error("Expected request to be aborted")
		}
	})
}

func TestOptionalAuth(t *testing.T) {
	// Setup gin test mode
	gin.SetMode(gin.TestMode)

	db := newMockDB()
	sessionManager := NewSessionManager(db)
	middleware := NewMiddleware(sessionManager)

	t.Run("no session continues without user", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)

		middleware.OptionalAuth()(c)

		if c.IsAborted() {
			t.Error("Expected request not to be aborted")
		}

		_, exists := GetUserFromContext(c)
		if exists {
			t.Error("User should not be in context when no session")
		}
	})

	t.Run("valid session adds user to context", func(t *testing.T) {
		user := &database.User{
			ID:       1,
			Email:    "test@example.com",
			Name:     "Test User",
			GoogleID: "google-123",
		}

		// Create a valid session
		session, err := sessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.AddCookie(&http.Cookie{
			Name:  "session_id_local",
			Value: session.ID,
		})

		middleware.OptionalAuth()(c)

		if c.IsAborted() {
			t.Error("Expected request not to be aborted")
		}

		retrievedUser, exists := GetUserFromContext(c)
		if !exists {
			t.Error("User not found in context")
		}
		if retrievedUser.ID != user.ID {
			t.Errorf("Wrong user in context: got %d, want %d", retrievedUser.ID, user.ID)
		}
	})
}

func TestRequireAdmin(t *testing.T) {
	// Setup gin test mode
	gin.SetMode(gin.TestMode)

	db := newMockDB()
	sessionManager := NewSessionManager(db)
	middleware := NewMiddleware(sessionManager)

	t.Run("no session returns unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/admin/users", nil)

		middleware.RequireAdmin()(c)

		if w.Code != 401 {
			t.Errorf("Expected status 401 (unauthorized), got %d", w.Code)
		}

		if !c.IsAborted() {
			t.Error("Expected request to be aborted")
		}
	})

	t.Run("non-admin user returns forbidden", func(t *testing.T) {
		user := &database.User{
			ID:       2,
			Email:    "test@example.com",
			Name:     "Test User",
			GoogleID: "google-123",
			IsAdmin:  false,
		}

		// Add user to mock DB so it can be retrieved
		db.users[user.ID] = user

		// Create a valid session
		session, err := sessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/admin/users", nil)
		c.Request.AddCookie(&http.Cookie{
			Name:  "session_id_local",
			Value: session.ID,
		})

		middleware.RequireAdmin()(c)

		if w.Code != 403 {
			t.Errorf("Expected status 403 (forbidden), got %d", w.Code)
		}

		if !c.IsAborted() {
			t.Error("Expected request to be aborted")
		}
	})

	t.Run("admin user adds user to context", func(t *testing.T) {
		user := &database.User{
			ID:       1,
			Email:    "admin@example.com",
			Name:     "Admin User",
			GoogleID: "google-456",
			IsAdmin:  true,
		}

		// Add user to mock DB so it can be retrieved
		db.users[user.ID] = user

		// Create a valid session
		session, err := sessionManager.CreateSession(user)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/admin/users", nil)
		c.Request.AddCookie(&http.Cookie{
			Name:  "session_id_local",
			Value: session.ID,
		})

		middleware.RequireAdmin()(c)

		if c.IsAborted() {
			t.Error("Expected request not to be aborted")
		}

		retrievedUser, exists := GetUserFromContext(c)
		if !exists {
			t.Error("User not found in context")
		}
		if retrievedUser.ID != user.ID {
			t.Errorf("Wrong user in context: got %d, want %d", retrievedUser.ID, user.ID)
		}
		if !retrievedUser.IsAdmin {
			t.Error("User should be admin")
		}
	})
}
