package helpers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"goread2/internal/auth"
	"goread2/internal/database"
	"goread2/internal/handlers"
	"goread2/internal/services"
)

// TestServer wraps the test server with authentication helpers
type TestServer struct {
	Router         *gin.Engine
	AuthService    *auth.AuthService
	SessionManager *auth.SessionManager
	FeedHandler    *handlers.FeedHandler
	AuthHandler    *handlers.AuthHandler
	DB             database.Database
}

// SetupTestServer creates a test server with all dependencies
func SetupTestServer(t *testing.T) *TestServer {
	gin.SetMode(gin.TestMode)

	db := CreateTestDB(t)

	// Create rate limiter for testing
	rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
		RequestsPerMinute: 60, // Higher rate for tests
		BurstSize:         10, // Allow some burst for tests
	})

	feedService := services.NewFeedService(db, rateLimiter)
	subscriptionService := services.NewSubscriptionService(db)
	authService := auth.NewAuthService(db)
	sessionManager := auth.NewSessionManager(db)

	// Create feed scheduler for testing (but don't start it)
	feedScheduler := services.NewFeedScheduler(feedService, rateLimiter, services.SchedulerConfig{
		UpdateWindow:    time.Hour,    // Shorter window for tests
		MinInterval:     time.Minute,  // Shorter interval for tests
		MaxConcurrent:   5,            // Fewer concurrent for tests
		CleanupInterval: 10 * time.Minute, // Less frequent cleanup for tests
	})

	feedHandler := handlers.NewFeedHandler(feedService, subscriptionService, feedScheduler)
	authHandler := handlers.NewAuthHandler(authService, sessionManager)
	authMiddleware := auth.NewMiddleware(sessionManager)

	router := gin.New()

	// Auth routes
	authGroup := router.Group("/auth")
	{
		authGroup.GET("/login", authHandler.Login)
		authGroup.GET("/callback", authHandler.Callback)
		authGroup.POST("/logout", authHandler.Logout)
		authGroup.GET("/me", authMiddleware.OptionalAuth(), authHandler.Me)
	}

	// Protected API routes
	api := router.Group("/api")
	api.Use(authMiddleware.RequireAuth())
	{
		api.GET("/feeds", feedHandler.GetFeeds)
		api.POST("/feeds", feedHandler.AddFeed)
		api.DELETE("/feeds/:id", feedHandler.DeleteFeed)
		api.GET("/feeds/:id/articles", feedHandler.GetArticles)
		api.POST("/articles/:id/read", feedHandler.MarkRead)
		api.POST("/articles/:id/star", feedHandler.ToggleStar)
		api.POST("/feeds/refresh", feedHandler.RefreshFeeds)
	}

	return &TestServer{
		Router:         router,
		AuthService:    authService,
		SessionManager: sessionManager,
		FeedHandler:    feedHandler,
		AuthHandler:    authHandler,
		DB:             db,
	}
}

// CreateAuthenticatedRequest creates an HTTP request with authentication
func (ts *TestServer) CreateAuthenticatedRequest(t *testing.T, method, url string, body interface{}, user *database.User) *http.Request {
	var req *http.Request
	var err error

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
	}

	// Create session for user
	session, err := ts.SessionManager.CreateSession(user)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Add session cookie
	cookie := &http.Cookie{
		Name:  "session_id",
		Value: session.ID,
	}
	req.AddCookie(cookie)

	return req
}

// CreateUnauthenticatedRequest creates an HTTP request without authentication
func CreateUnauthenticatedRequest(t *testing.T, method, url string, body interface{}) *http.Request {
	var req *http.Request
	var err error

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
	}

	return req
}

// ExecuteRequest executes an HTTP request and returns the response
func (ts *TestServer) ExecuteRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	ts.Router.ServeHTTP(rr, req)
	return rr
}

// AssertJSONResponse asserts that the response has the expected status code and JSON body
func AssertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedBody interface{}) {
	if rr.Code != expectedStatus {
		t.Errorf("Expected status code %d, got %d. Response body: %s", expectedStatus, rr.Code, rr.Body.String())
	}

	if expectedBody != nil {
		var actualBody interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &actualBody); err != nil {
			t.Fatalf("Failed to unmarshal response body: %v", err)
		}

		expectedJSON, _ := json.Marshal(expectedBody)
		actualJSON, _ := json.Marshal(actualBody)

		if !bytes.Equal(expectedJSON, actualJSON) {
			t.Errorf("Expected JSON body %s, got %s", expectedJSON, actualJSON)
		}
	}
}
