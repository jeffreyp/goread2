package helpers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/handlers"
	"github.com/jeffreyp/goread2/internal/services"
)

// TestServer wraps the test server with authentication helpers
type TestServer struct {
	Router         *gin.Engine
	AuthService    *auth.AuthService
	SessionManager *auth.SessionManager
	CSRFManager    *auth.CSRFManager
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
		UpdateWindow:    2 * time.Second,        // Very short window for fast tests
		MinInterval:     100 * time.Millisecond, // Minimal interval for tests
		MaxConcurrent:   10,                     // More concurrent for faster tests
		CleanupInterval: 10 * time.Minute,       // Less frequent cleanup for tests
	})

	csrfManager := auth.NewCSRFManager()
	feedHandler := handlers.NewFeedHandler(feedService, subscriptionService, feedScheduler, db)
	authHandler := handlers.NewAuthHandler(authService, sessionManager, csrfManager)
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
	api.Use(authMiddleware.CSRFMiddleware(csrfManager)) // Enable CSRF protection in tests
	{
		api.GET("/feeds", feedHandler.GetFeeds)
		api.POST("/feeds", feedHandler.AddFeed)
		api.DELETE("/feeds/:id", feedHandler.DeleteFeed)
		api.POST("/feeds/import", feedHandler.ImportOPML)
		api.GET("/feeds/export", feedHandler.ExportOPML)
		api.GET("/feeds/:id/articles", feedHandler.GetArticles)
		api.POST("/articles/:id/read", feedHandler.MarkRead)
		api.POST("/articles/:id/star", feedHandler.ToggleStar)
		api.POST("/articles/mark-all-read", feedHandler.MarkAllRead)
		api.POST("/feeds/refresh", feedHandler.RefreshFeeds)
	}

	return &TestServer{
		Router:         router,
		AuthService:    authService,
		SessionManager: sessionManager,
		CSRFManager:    csrfManager,
		FeedHandler:    feedHandler,
		AuthHandler:    authHandler,
		DB:             db,
	}
}

// CreateAuthenticatedRequest creates an HTTP request with authentication and CSRF token
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
	// Tests run in local mode, so use the local cookie name
	cookie := &http.Cookie{
		Name:  "session_id_local",
		Value: session.ID,
	}
	req.AddCookie(cookie)

	// Add CSRF token for state-changing methods (POST, PUT, DELETE, PATCH)
	if method != "GET" && method != "HEAD" && method != "OPTIONS" {
		csrfToken, err := ts.CSRFManager.GenerateToken(session.ID)
		if err != nil {
			t.Fatalf("Failed to generate CSRF token: %v", err)
		}
		req.Header.Set("X-CSRF-Token", csrfToken)
	}

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

// NewMockFeedServer creates a mock HTTP server that serves RSS/Atom feed content
// Returns the server and its URL. The caller must call server.Close() when done.
func NewMockFeedServer(t *testing.T, feedXML string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(feedXML))
	}))
}

// NewMockFeedServerWithStatus creates a mock HTTP server that returns a specific status code
func NewMockFeedServerWithStatus(t *testing.T, statusCode int, body string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	}))
}

// NewMockMultiFeedServer creates a mock HTTP server that can serve different feeds based on the path
func NewMockMultiFeedServer(t *testing.T, feeds map[string]string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		feedXML, exists := feeds[r.URL.Path]
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(feedXML))
	}))
}

// MockHTTPClient wraps an httptest.Server to provide an HTTP client
// that redirects all requests to the mock server
type MockHTTPClient struct {
	Server *httptest.Server
}

// Do implements the HTTPClient interface by executing the request
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Use the server's client to make the request
	return m.Server.Client().Do(req)
}

// NewMockHTTPClient creates a mock HTTP client from an httptest.Server
func NewMockHTTPClient(server *httptest.Server) *MockHTTPClient {
	return &MockHTTPClient{
		Server: server,
	}
}
