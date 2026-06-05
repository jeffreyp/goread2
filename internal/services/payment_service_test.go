package services

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
	"github.com/stripe/stripe-go/v78"
)

// newTestPaymentService creates a PaymentService with controlled credentials.
// Uses struct literal since tests are in the same package.
func newTestPaymentService(db database.Database, pubKey, secretKey, webhookSecret, priceID string) *PaymentService {
	return &PaymentService{
		db:                   db,
		stripePublishableKey: pubKey,
		stripeSecretKey:      secretKey,
		stripeWebhookSecret:  webhookSecret,
		stripePriceID:        priceID,
	}
}

// setStripeTestBackend points the Stripe SDK at a local test server and returns a cleanup func.
func setStripeTestBackend(server *httptest.Server) func() {
	backends := stripe.NewBackends(server.Client())
	// Override the API URL so requests go to the test server.
	apiBackend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
		URL:        stripe.String(server.URL),
		HTTPClient: server.Client(),
	})
	stripe.SetBackend(stripe.APIBackend, apiBackend)
	_ = backends // suppress unused warning
	return func() {
		// Restore default backend.
		stripe.SetBackend(stripe.APIBackend, stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{}))
	}
}

// mockStripeServer creates an httptest.Server that returns JSON responses for Stripe API calls.
func mockStripeServer(t *testing.T, handlers map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for path, resp := range handlers {
			if r.URL.Path == path {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
		}
		// Default: return an empty object so Stripe SDK doesn't panic.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
}

// mockDBPayment is a minimal mock for payment service tests.
type mockDBPayment struct {
	user          *database.User
	shouldFailGet bool
	updateCalled  bool
}

func (m *mockDBPayment) GetUserByID(int) (*database.User, error) {
	if m.shouldFailGet {
		return nil, errors.New("user not found")
	}
	if m.user != nil {
		return m.user, nil
	}
	return &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}, nil
}
func (m *mockDBPayment) UpdateUserSubscription(userID int, status, subscriptionID string, lastPaymentDate, nextBillingDate time.Time) error {
	m.updateCalled = true
	return nil
}
func (m *mockDBPayment) Close() error                                                         { return nil }
func (m *mockDBPayment) CreateUser(*database.User) error                                      { return nil }
func (m *mockDBPayment) GetUserByGoogleID(string) (*database.User, error)                     { return nil, nil }
func (m *mockDBPayment) IsUserSubscriptionActive(int) (bool, error)                           { return false, nil }
func (m *mockDBPayment) GetUserFeedCount(int) (int, error)                                    { return 0, nil }
func (m *mockDBPayment) UpdateUserMaxArticlesOnFeedAdd(int, int) error                        { return nil }
func (m *mockDBPayment) SetUserAdmin(int, bool) error                                         { return nil }
func (m *mockDBPayment) SetUserAdminAtomic(int, int, bool) error                              { return nil }
func (m *mockDBPayment) GrantFreeMonths(int, int) error                                       { return nil }
func (m *mockDBPayment) GetUserByEmail(string) (*database.User, error)                        { return nil, nil }
func (m *mockDBPayment) AddFeed(*database.Feed) error                                         { return nil }
func (m *mockDBPayment) UpdateFeed(*database.Feed) error                                      { return nil }
func (m *mockDBPayment) UpdateFeedTracking(int, time.Time, time.Time, int) error              { return nil }
func (m *mockDBPayment) GetFeeds() ([]database.Feed, error)                                   { return nil, nil }
func (m *mockDBPayment) GetFeedByURL(string) (*database.Feed, error)                          { return nil, nil }
func (m *mockDBPayment) GetUserFeeds(int) ([]database.Feed, error)                            { return nil, nil }
func (m *mockDBPayment) GetAllUserFeeds() ([]database.Feed, error)                            { return nil, nil }
func (m *mockDBPayment) UpdateFeedCacheHeaders(int, string, string) error                     { return nil }
func (m *mockDBPayment) DeleteFeed(int) error                                                 { return nil }
func (m *mockDBPayment) SubscribeUserToFeed(int, int) error                                   { return nil }
func (m *mockDBPayment) UnsubscribeUserFromFeed(int, int) error                               { return nil }
func (m *mockDBPayment) AddArticle(*database.Article) error                                   { return nil }
func (m *mockDBPayment) FilterExistingArticleURLs(int, []string) (map[string]bool, error)     { return nil, nil }
func (m *mockDBPayment) GetArticles(int) ([]database.Article, error)                          { return nil, nil }
func (m *mockDBPayment) FindArticleByURL(string) (*database.Article, error)                   { return nil, nil }
func (m *mockDBPayment) GetUserArticles(int) ([]database.Article, error)                      { return nil, nil }
func (m *mockDBPayment) GetUserArticlesPaginated(int, int, string, bool) (*database.ArticlePaginationResult, error) {
	return &database.ArticlePaginationResult{}, nil
}
func (m *mockDBPayment) GetUserFeedArticles(int, int) ([]database.Article, error)             { return nil, nil }
func (m *mockDBPayment) GetArticleByID(int, int) (*database.Article, error)                   { return nil, nil }
func (m *mockDBPayment) GetUserArticleStatus(int, int) (*database.UserArticle, error)         { return nil, nil }
func (m *mockDBPayment) SetUserArticleStatus(int, int, bool, bool) error                      { return nil }
func (m *mockDBPayment) MarkAllUserArticlesRead(int) (int, error)                             { return 0, nil }
func (m *mockDBPayment) BatchSetUserArticleStatus(int, []database.Article, bool, bool) error  { return nil }
func (m *mockDBPayment) MarkUserArticleRead(int, int, bool) error                             { return nil }
func (m *mockDBPayment) ToggleUserArticleStar(int, int) error                                 { return nil }
func (m *mockDBPayment) GetUserUnreadCounts(int) (map[int]int, error)                         { return nil, nil }
func (m *mockDBPayment) GetTotalArticleCount(int) (int, error)                                { return 0, nil }
func (m *mockDBPayment) GetAccountStats(int) (map[string]interface{}, error)                  { return nil, nil }
func (m *mockDBPayment) CleanupOrphanedUserArticles(int) (int, error)                         { return 0, nil }
func (m *mockDBPayment) CreateSession(*database.Session) error                                { return nil }
func (m *mockDBPayment) GetSession(string) (*database.Session, error)                         { return nil, nil }
func (m *mockDBPayment) UpdateSessionExpiry(string, time.Time) error                          { return nil }
func (m *mockDBPayment) DeleteSession(string) error                                           { return nil }
func (m *mockDBPayment) DeleteExpiredSessions() error                                         { return nil }
func (m *mockDBPayment) CreateAuditLog(*database.AuditLog) error                              { return nil }
func (m *mockDBPayment) GetAuditLogs(int, int, map[string]interface{}) ([]database.AuditLog, error) {
	return nil, nil
}
func (m *mockDBPayment) UpdateFeedLastFetch(int, time.Time) error { return nil }
func (m *mockDBPayment) UpdateFeedAfterRefresh(int, time.Time, time.Time, int, time.Time, string, string) error {
	return nil
}

// --- Tests ---

func TestPaymentService_GetStripePublishableKey(t *testing.T) {
	ps := newTestPaymentService(nil, "pk_test_abc", "", "", "")
	if got := ps.GetStripePublishableKey(); got != "pk_test_abc" {
		t.Errorf("expected 'pk_test_abc', got %q", got)
	}
}

func TestPaymentService_GetStripeWebhookSecret(t *testing.T) {
	ps := newTestPaymentService(nil, "", "", "whsec_test123", "")
	if got := ps.GetStripeWebhookSecret(); got != "whsec_test123" {
		t.Errorf("expected 'whsec_test123', got %q", got)
	}
}

func TestPaymentService_ValidateStripeConfig_MissingKeys(t *testing.T) {
	ps := newTestPaymentService(nil, "", "", "", "")
	if err := ps.ValidateStripeConfig(); err == nil {
		t.Error("expected error when all keys are empty")
	}
}

func TestPaymentService_ValidateStripeConfig_Valid(t *testing.T) {
	ps := newTestPaymentService(nil, "pk_test", "sk_test", "whsec_test", "price_test")
	if err := ps.ValidateStripeConfig(); err != nil {
		t.Errorf("expected no error with all keys set, got: %v", err)
	}
}

func TestPaymentService_CreateCheckoutSession_ErrAlreadySubscribed(t *testing.T) {
	db := &mockDBPayment{
		user: &database.User{
			ID: 1, Email: "user@example.com",
			SubscriptionStatus: "active",
		},
	}
	ps := newTestPaymentService(db, "pk", "sk", "wh", "price")

	_, err := ps.CreateCheckoutSession(CheckoutSessionRequest{
		UserID: 1, SuccessURL: "https://example.com/success", CancelURL: "https://example.com/cancel",
	})
	if !errors.Is(err, ErrAlreadySubscribed) {
		t.Errorf("expected ErrAlreadySubscribed, got: %v", err)
	}
}

func TestPaymentService_CreateCheckoutSession_UserNotFound(t *testing.T) {
	db := &mockDBPayment{shouldFailGet: true}
	ps := newTestPaymentService(db, "pk", "sk", "wh", "price")

	_, err := ps.CreateCheckoutSession(CheckoutSessionRequest{UserID: 99})
	if err == nil {
		t.Error("expected error when user not found")
	}
}

func TestPaymentService_CreateCheckoutSession_HappyPath(t *testing.T) {
	// Mock Stripe endpoints needed for customer search and checkout session creation.
	srv := mockStripeServer(t, map[string]interface{}{
		"/v1/customers/search": map[string]interface{}{
			"object": "search_result", "data": []interface{}{},
			"has_more": false,
		},
		"/v1/customers": map[string]interface{}{
			"id": "cus_test123", "object": "customer",
		},
		"/v1/checkout/sessions": map[string]interface{}{
			"id": "cs_test_abc", "object": "checkout.session",
			"url": "https://checkout.stripe.com/pay/cs_test_abc",
		},
	})
	defer srv.Close()
	restoreBackend := setStripeTestBackend(srv)
	defer restoreBackend()

	stripe.Key = "sk_test_fake"

	db := &mockDBPayment{
		user: &database.User{ID: 1, Email: "user@example.com", Name: "Test User"},
	}
	ps := newTestPaymentService(db, "pk_test", "sk_test_fake", "whsec_test", "price_test")
	ps.stripePriceID = "price_test123"

	resp, err := ps.CreateCheckoutSession(CheckoutSessionRequest{
		UserID: 1, SuccessURL: "https://example.com/success", CancelURL: "https://example.com/cancel",
	})
	if err != nil {
		t.Fatalf("CreateCheckoutSession: %v", err)
	}
	if resp.SessionID != "cs_test_abc" {
		t.Errorf("expected session ID 'cs_test_abc', got %q", resp.SessionID)
	}
}

func TestPaymentService_CreateCustomerPortalSession_UserNotFound(t *testing.T) {
	db := &mockDBPayment{shouldFailGet: true}
	ps := newTestPaymentService(db, "pk", "sk", "wh", "price")

	_, err := ps.CreateCustomerPortalSession(99, "https://example.com/return")
	if err == nil {
		t.Error("expected error when user not found")
	}
}

func TestPaymentService_CreateCustomerPortalSession_HappyPath(t *testing.T) {
	srv := mockStripeServer(t, map[string]interface{}{
		"/v1/customers/search": map[string]interface{}{
			"object": "search_result",
			"data":   []interface{}{map[string]interface{}{"id": "cus_existing", "object": "customer"}},
			"has_more": false,
		},
		"/v1/billing_portal/sessions": map[string]interface{}{
			"id": "bps_test", "object": "billing_portal.session",
			"url": "https://billing.stripe.com/p/session/test",
		},
	})
	defer srv.Close()
	restoreBackend := setStripeTestBackend(srv)
	defer restoreBackend()

	stripe.Key = "sk_test_fake"

	db := &mockDBPayment{
		user: &database.User{ID: 1, Email: "user@example.com", Name: "Test User"},
	}
	ps := newTestPaymentService(db, "pk_test", "sk_test_fake", "whsec_test", "price_test")

	portalURL, err := ps.CreateCustomerPortalSession(1, "https://example.com/return")
	if err != nil {
		t.Fatalf("CreateCustomerPortalSession: %v", err)
	}
	if portalURL == "" {
		t.Error("expected non-empty portal URL")
	}
}
