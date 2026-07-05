package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/services"
	"github.com/stripe/stripe-go/v78/webhook"
)

// mockPaymentService implements paymentServicer for unit tests.
type mockPaymentService struct {
	publishableKey  string
	webhookSecret   string
	checkoutSession *services.CheckoutSessionResponse
	checkoutErr     error
	portalURL       string
	portalErr       error
	subscriptionErr error
}

func (m *mockPaymentService) CreateCheckoutSession(req services.CheckoutSessionRequest) (*services.CheckoutSessionResponse, error) {
	return m.checkoutSession, m.checkoutErr
}
func (m *mockPaymentService) GetStripePublishableKey() string { return m.publishableKey }
func (m *mockPaymentService) GetStripeWebhookSecret() string  { return m.webhookSecret }
func (m *mockPaymentService) HandleSubscriptionUpdate(subscriptionID string) error {
	return m.subscriptionErr
}
func (m *mockPaymentService) CreateCustomerPortalSession(userID int, returnURL string) (string, error) {
	return m.portalURL, m.portalErr
}

// newPaymentHandlerWithMock creates a PaymentHandler backed by the mock service.
func newPaymentHandlerWithMock(svc paymentServicer) *PaymentHandler {
	return &PaymentHandler{paymentService: svc, baseURL: "https://example.com"}
}

func TestNewPaymentHandler(t *testing.T) {
	handler := NewPaymentHandler(&services.PaymentService{}, "https://example.com/auth/callback")

	if handler == nil {
		t.Fatal("NewPaymentHandler returned nil")
	}
	if handler.baseURL != "https://example.com" {
		t.Errorf("baseURL not extracted correctly: got %q", handler.baseURL)
	}
}

func TestGetStripeConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPaymentService{publishableKey: "pk_test_abc123"}
	handler := newPaymentHandlerWithMock(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/stripe-config", nil)

	handler.GetStripeConfig(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["publishable_key"] != "pk_test_abc123" {
		t.Errorf("expected publishable_key 'pk_test_abc123', got %q", resp["publishable_key"])
	}
}

func TestCreateCheckoutSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		handler := newPaymentHandlerWithMock(&mockPaymentService{})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/subscription/checkout", nil)

		handler.CreateCheckoutSession(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("already subscribed returns 409", func(t *testing.T) {
		svc := &mockPaymentService{checkoutErr: services.ErrAlreadySubscribed}
		handler := newPaymentHandlerWithMock(svc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/subscription/checkout", nil)
		c.Set("user", testUser)

		handler.CreateCheckoutSession(c)

		if w.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", w.Code)
		}
		var resp map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["error"] == "" {
			t.Error("expected error message in response")
		}
	})

	t.Run("happy path returns 200 with session", func(t *testing.T) {
		svc := &mockPaymentService{
			checkoutSession: &services.CheckoutSessionResponse{
				SessionID:  "cs_test_abc",
				SessionURL: "https://checkout.stripe.com/pay/cs_test_abc",
			},
		}
		handler := newPaymentHandlerWithMock(svc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/subscription/checkout", nil)
		c.Set("user", testUser)

		handler.CreateCheckoutSession(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp services.CheckoutSessionResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp.SessionID != "cs_test_abc" {
			t.Errorf("expected session_id 'cs_test_abc', got %q", resp.SessionID)
		}
	})
}

func TestCreateCustomerPortal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testUser := &database.User{ID: 1, Email: "test@example.com", Name: "Test User"}

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		handler := newPaymentHandlerWithMock(&mockPaymentService{})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/subscription/portal", nil)

		handler.CreateCustomerPortal(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("happy path returns portal URL", func(t *testing.T) {
		svc := &mockPaymentService{portalURL: "https://billing.stripe.com/p/session/test"}
		handler := newPaymentHandlerWithMock(svc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/subscription/portal", nil)
		c.Set("user", testUser)

		handler.CreateCustomerPortal(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["portal_url"] != "https://billing.stripe.com/p/session/test" {
			t.Errorf("unexpected portal_url: %q", resp["portal_url"])
		}
	})
}

func TestWebhookHandler_MissingSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPaymentService{webhookSecret: ""}
	handler := newPaymentHandlerWithMock(svc)

	payload := []byte(`{"id":"evt_test","type":"customer.subscription.created","data":{"object":{"id":"sub_test"}}}`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(payload))
	c.Request.Header.Set("Stripe-Signature", "t=1,v1=fake")

	handler.WebhookHandler(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when webhook secret missing, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("not properly configured")) {
		t.Errorf("expected 'not properly configured' error, got: %s", w.Body.String())
	}
}

func TestWebhookHandler_InvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPaymentService{webhookSecret: "whsec_testsecret123"}
	handler := newPaymentHandlerWithMock(svc)

	payload := []byte(`{"id":"evt_test","type":"customer.subscription.created","data":{"object":{"id":"sub_test"}}}`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(payload))
	c.Request.Header.Set("Stripe-Signature", "t=1,v1=invalidsignature")

	handler.WebhookHandler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid signature, got %d", w.Code)
	}
}

// signedWebhookRequest builds a Stripe-signed POST request for the given
// event payload, using the real stripe-go signing helper so ConstructEventWithOptions
// verifies successfully against secret.
func signedWebhookRequest(t *testing.T, secret string, payload []byte) *http.Request {
	t.Helper()
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload: payload,
		Secret:  secret,
	})
	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signed.Header)
	return req
}

func TestWebhookHandler_CheckoutSessionCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "whsec_testsecret123"
	svc := &mockPaymentService{webhookSecret: secret}
	handler := newPaymentHandlerWithMock(svc)

	payload := []byte(`{"id":"evt_test","type":"checkout.session.completed","data":{"object":{"id":"cs_test_123","mode":"subscription"}}}`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = signedWebhookRequest(t, secret, payload)

	handler.WebhookHandler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWebhookHandler_SubscriptionCreatedOrUpdated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "whsec_testsecret123"

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockPaymentService{webhookSecret: secret}
		handler := newPaymentHandlerWithMock(svc)
		payload := []byte(`{"id":"evt_test","type":"customer.subscription.created","data":{"object":{"id":"sub_test123","status":"active"}}}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = signedWebhookRequest(t, secret, payload)

		handler.WebhookHandler(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("subscription.updated success returns 200", func(t *testing.T) {
		svc := &mockPaymentService{webhookSecret: secret}
		handler := newPaymentHandlerWithMock(svc)
		payload := []byte(`{"id":"evt_test","type":"customer.subscription.updated","data":{"object":{"id":"sub_test123","status":"active"}}}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = signedWebhookRequest(t, secret, payload)

		handler.WebhookHandler(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockPaymentService{webhookSecret: secret, subscriptionErr: errors.New("db error")}
		handler := newPaymentHandlerWithMock(svc)
		payload := []byte(`{"id":"evt_test","type":"customer.subscription.created","data":{"object":{"id":"sub_test123","status":"active"}}}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = signedWebhookRequest(t, secret, payload)

		handler.WebhookHandler(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("malformed subscription object returns 400", func(t *testing.T) {
		svc := &mockPaymentService{webhookSecret: secret}
		handler := newPaymentHandlerWithMock(svc)
		payload := []byte(`{"id":"evt_test","type":"customer.subscription.created","data":{"object":"not-an-object"}}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = signedWebhookRequest(t, secret, payload)

		handler.WebhookHandler(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestWebhookHandler_SubscriptionDeleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "whsec_testsecret123"

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockPaymentService{webhookSecret: secret}
		handler := newPaymentHandlerWithMock(svc)
		payload := []byte(`{"id":"evt_test","type":"customer.subscription.deleted","data":{"object":{"id":"sub_test123","status":"canceled"}}}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = signedWebhookRequest(t, secret, payload)

		handler.WebhookHandler(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockPaymentService{webhookSecret: secret, subscriptionErr: errors.New("db error")}
		handler := newPaymentHandlerWithMock(svc)
		payload := []byte(`{"id":"evt_test","type":"customer.subscription.deleted","data":{"object":{"id":"sub_test123","status":"canceled"}}}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = signedWebhookRequest(t, secret, payload)

		handler.WebhookHandler(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("malformed subscription object returns 400", func(t *testing.T) {
		svc := &mockPaymentService{webhookSecret: secret}
		handler := newPaymentHandlerWithMock(svc)
		payload := []byte(`{"id":"evt_test","type":"customer.subscription.deleted","data":{"object":"not-an-object"}}`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = signedWebhookRequest(t, secret, payload)

		handler.WebhookHandler(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestWebhookHandler_UnhandledEventType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "whsec_testsecret123"
	svc := &mockPaymentService{webhookSecret: secret}
	handler := newPaymentHandlerWithMock(svc)

	payload := []byte(`{"id":"evt_test","type":"invoice.paid","data":{"object":{"id":"in_test123"}}}`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = signedWebhookRequest(t, secret, payload)

	handler.WebhookHandler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWebhookHandler_BodyTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPaymentService{webhookSecret: "whsec_testsecret123"}
	handler := newPaymentHandlerWithMock(svc)

	oversized := bytes.Repeat([]byte("a"), 70000)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(oversized))
	c.Request.Header.Set("Stripe-Signature", "t=1,v1=fake")

	handler.WebhookHandler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for oversized body, got %d: %s", w.Code, w.Body.String())
	}
}

func newHTMLTestEngine(t *testing.T, name, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	tmpl := template.Must(template.New(name).Parse(body))
	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.SetHTMLTemplate(tmpl)
	return c, w
}

func TestSubscriptionSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := newPaymentHandlerWithMock(&mockPaymentService{})
	c, w := newHTMLTestEngine(t, "subscription_success.html", `{{define "subscription_success.html"}}session={{.session_id}}{{end}}`)
	c.Request = httptest.NewRequest("GET", "/subscription/success?session_id=cs_test_abc", nil)

	handler.SubscriptionSuccess(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("cs_test_abc")) {
		t.Errorf("expected session_id in rendered body, got: %s", w.Body.String())
	}
}

func TestSubscriptionCancel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := newPaymentHandlerWithMock(&mockPaymentService{})
	c, w := newHTMLTestEngine(t, "subscription_cancel.html", `{{define "subscription_cancel.html"}}cancelled{{end}}`)
	c.Request = httptest.NewRequest("GET", "/subscription/cancel", nil)

	handler.SubscriptionCancel(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("cancelled")) {
		t.Errorf("expected rendered body, got: %s", w.Body.String())
	}
}
