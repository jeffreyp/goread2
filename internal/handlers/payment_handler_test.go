package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/services"
)

func TestNewPaymentHandler(t *testing.T) {
	// Create mock service
	mockPaymentService := &services.PaymentService{}

	handler := NewPaymentHandler(mockPaymentService)

	if handler == nil {
		t.Fatal("NewPaymentHandler returned nil")
	}

	if handler.paymentService != mockPaymentService {
		t.Error("PaymentHandler payment service not set correctly")
	}
}

func TestWebhookHandler_MissingSecret(t *testing.T) {
	// Set up test environment
	gin.SetMode(gin.TestMode)

	// Create payment service with empty webhook secret
	// This simulates the security vulnerability where STRIPE_WEBHOOK_SECRET is not set
	// The PaymentService's GetStripeWebhookSecret() will return an empty string
	paymentService := &services.PaymentService{}

	handler := NewPaymentHandler(paymentService)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a fake Stripe webhook payload
	payload := []byte(`{
		"id": "evt_test",
		"type": "customer.subscription.created",
		"data": {
			"object": {
				"id": "sub_test"
			}
		}
	}`)
	c.Request = httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Stripe-Signature", "t=1,v1=fake_signature")

	// Call the webhook handler
	handler.WebhookHandler(c)

	// Verify that the request was rejected with 500 Internal Server Error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d when webhook secret is missing, got %d", http.StatusInternalServerError, w.Code)
	}

	// Verify error message contains indication that webhook is not configured
	if !bytes.Contains(w.Body.Bytes(), []byte("not properly configured")) {
		t.Errorf("Expected error message about webhook not being configured, got: %s", w.Body.String())
	}
}

func TestWebhookHandler_InvalidSignature(t *testing.T) {
	// This test would require mocking the Stripe webhook verification
	// Since webhook.ConstructEventWithOptions validates against a real secret,
	// we'd need to either:
	// 1. Use a real Stripe test secret and generate a valid signature
	// 2. Mock the webhook verification layer
	// For now, we'll skip this test as it requires integration with Stripe's signing

	t.Skip("Skipping webhook signature validation test - requires Stripe integration or mocking")
}
