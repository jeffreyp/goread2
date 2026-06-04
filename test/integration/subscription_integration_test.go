package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/config"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/handlers"
	"github.com/jeffreyp/goread2/internal/secrets"
	"github.com/jeffreyp/goread2/internal/services"
	"github.com/jeffreyp/goread2/test/helpers"
)

// subscriptionTestServer builds a gin router with subscription-related routes.
// paymentHandler may be nil — in that case payment routes are not registered.
func subscriptionTestServer(
	t *testing.T,
	db database.Database,
	sessionManager *auth.SessionManager,
	csrfManager *auth.CSRFManager,
	feedHandler *handlers.FeedHandler,
	paymentHandler *handlers.PaymentHandler,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	authMiddleware := auth.NewMiddleware(sessionManager)
	router := gin.New()

	api := router.Group("/api")
	api.Use(authMiddleware.RequireAuth())
	api.Use(authMiddleware.CSRFMiddleware(csrfManager))
	{
		api.GET("/subscription", feedHandler.GetSubscriptionInfo)

		if paymentHandler != nil {
			api.POST("/subscription/checkout", paymentHandler.CreateCheckoutSession)
			api.POST("/subscription/portal", paymentHandler.CreateCustomerPortal)
		}
	}

	if paymentHandler != nil {
		router.POST("/webhooks/stripe", paymentHandler.WebhookHandler)
	}

	return router
}

// makeAuthenticatedRequest creates an authenticated request for subscription tests.
func makeAuthRequest(t *testing.T, method, url string, body interface{}, user *database.User, sm *auth.SessionManager, csrfMgr *auth.CSRFManager) *http.Request {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
	}
	var req *http.Request
	if bodyBytes != nil {
		req, _ = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}

	session, err := sm.CreateSession(user)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: "session_id_local", Value: session.ID})

	if method != "GET" && method != "HEAD" && method != "OPTIONS" {
		token, err := csrfMgr.GenerateToken(session.ID)
		if err != nil {
			t.Fatalf("failed to generate CSRF token: %v", err)
		}
		req.Header.Set("X-CSRF-Token", token)
	}
	return req
}

func TestSubscriptionStatusEndpoint(t *testing.T) {
	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	db := helpers.CreateTestDB(t)
	rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{RequestsPerMinute: 60, BurstSize: 10})
	feedService := services.NewFeedService(db, rateLimiter)
	subscriptionService := services.NewSubscriptionService(db)
	sessionManager := auth.NewSessionManager(db)
	csrfManager := auth.NewCSRFManager()
	feedScheduler := services.NewFeedScheduler(feedService, rateLimiter, services.SchedulerConfig{
		UpdateWindow:    2 * time.Second,
		MinInterval:     100 * time.Millisecond,
		MaxConcurrent:   2,
		CleanupInterval: 10 * time.Minute,
	})
	feedHandler := handlers.NewFeedHandler(feedService, subscriptionService, feedScheduler, db)
	router := subscriptionTestServer(t, db, sessionManager, csrfManager, feedHandler, nil)

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/subscription", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("free user returns subscription info", func(t *testing.T) {
		user := helpers.CreateTestUser(t, db, "sub-free-001", "subfree@example.com", "Free User")

		req := makeAuthRequest(t, "GET", "/api/subscription", nil, user, sessionManager, csrfManager)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var info map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &info); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if info["status"] == nil {
			t.Error("expected 'status' field in subscription info")
		}
	})

	t.Run("trial user returns correct status", func(t *testing.T) {
		user := helpers.CreateTestUser(t, db, "sub-trial-001", "subtrial@example.com", "Trial User")

		req := makeAuthRequest(t, "GET", "/api/subscription", nil, user, sessionManager, csrfManager)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestSubscriptionPaymentEndpointsRequireAuth(t *testing.T) {
	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	db := helpers.CreateTestDB(t)
	rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{RequestsPerMinute: 60, BurstSize: 10})
	feedService := services.NewFeedService(db, rateLimiter)
	subscriptionService := services.NewSubscriptionService(db)
	sessionManager := auth.NewSessionManager(db)
	csrfManager := auth.NewCSRFManager()
	feedScheduler := services.NewFeedScheduler(feedService, rateLimiter, services.SchedulerConfig{
		UpdateWindow:    2 * time.Second,
		MinInterval:     100 * time.Millisecond,
		MaxConcurrent:   2,
		CleanupInterval: 10 * time.Minute,
	})
	feedHandler := handlers.NewFeedHandler(feedService, subscriptionService, feedScheduler, db)
	paymentHandler := handlers.NewPaymentHandler(nil, "https://example.com/auth/callback")
	router := subscriptionTestServer(t, db, sessionManager, csrfManager, feedHandler, paymentHandler)

	endpoints := []struct {
		method string
		path   string
	}{
		{"POST", "/api/subscription/checkout"},
		{"POST", "/api/subscription/portal"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path+" unauthenticated returns 401", func(t *testing.T) {
			req, _ := http.NewRequest(ep.method, ep.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", rr.Code)
			}
		})
	}
}

func TestWebhookRejectsInvalidSignature(t *testing.T) {
	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	// Provide all Stripe env vars before creating the PaymentService so the
	// secrets.sync.Once succeeds without contacting Google Secret Manager.
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_fake_for_testing")
	t.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_fake_for_testing")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_only_not_a_real_secret")
	t.Setenv("STRIPE_PRICE_ID", "price_test_fake_for_testing")
	secrets.ResetCacheForTesting()
	t.Cleanup(secrets.ResetCacheForTesting)

	db := helpers.CreateTestDB(t)
	rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{RequestsPerMinute: 60, BurstSize: 10})
	feedService := services.NewFeedService(db, rateLimiter)
	subscriptionService := services.NewSubscriptionService(db)
	sessionManager := auth.NewSessionManager(db)
	csrfManager := auth.NewCSRFManager()
	feedScheduler := services.NewFeedScheduler(feedService, rateLimiter, services.SchedulerConfig{
		UpdateWindow:    2 * time.Second,
		MinInterval:     100 * time.Millisecond,
		MaxConcurrent:   2,
		CleanupInterval: 10 * time.Minute,
	})
	feedHandler := handlers.NewFeedHandler(feedService, subscriptionService, feedScheduler, db)
	paymentHandler := handlers.NewPaymentHandler(
		services.NewPaymentService(db, subscriptionService),
		"https://example.com/auth/callback",
	)
	router := subscriptionTestServer(t, db, sessionManager, csrfManager, feedHandler, paymentHandler)

	payload := []byte(`{"id":"evt_test","type":"customer.subscription.created","data":{"object":{"id":"sub_test"}}}`)

	req, _ := http.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "t=1234567890,v1=invalidsignature")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid webhook signature, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestWebhookMissingSecret is covered at the unit level in payment_handler_test.go.
// The secrets package uses sync.Once so the missing-secret case cannot be reliably
// tested at integration level once real credentials have been cached.

func TestSubscriptionFeatureFlag_RoutesAbsentWhenDisabled(t *testing.T) {
	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	defer func() {
		_ = os.Unsetenv("SUBSCRIPTION_ENABLED")
		config.ResetForTesting()
	}()

	_ = os.Setenv("SUBSCRIPTION_ENABLED", "false")
	config.ResetForTesting()
	config.Load()

	db := helpers.CreateTestDB(t)
	rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{RequestsPerMinute: 60, BurstSize: 10})
	feedService := services.NewFeedService(db, rateLimiter)
	subscriptionService := services.NewSubscriptionService(db)
	sessionManager := auth.NewSessionManager(db)
	csrfManager := auth.NewCSRFManager()
	feedScheduler := services.NewFeedScheduler(feedService, rateLimiter, services.SchedulerConfig{
		UpdateWindow:    2 * time.Second,
		MinInterval:     100 * time.Millisecond,
		MaxConcurrent:   2,
		CleanupInterval: 10 * time.Minute,
	})
	feedHandler := handlers.NewFeedHandler(feedService, subscriptionService, feedScheduler, db)

	// When subscription is disabled, pass nil paymentHandler — routes not registered.
	router := subscriptionTestServer(t, db, sessionManager, csrfManager, feedHandler, nil)

	endpoints := []string{
		"/api/subscription/checkout",
		"/api/subscription/portal",
		"/webhooks/stripe",
	}
	for _, path := range endpoints {
		t.Run("route not registered: "+path, func(t *testing.T) {
			req, _ := http.NewRequest("POST", path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusNotFound {
				t.Errorf("expected 404 for %s when subscription disabled, got %d", path, rr.Code)
			}
		})
	}
}
