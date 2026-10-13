package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/services"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
)

type paymentServicer interface {
	CreateCheckoutSession(req services.CheckoutSessionRequest) (*services.CheckoutSessionResponse, error)
	GetStripePublishableKey() string
	GetStripeWebhookSecret() string
	HandleSubscriptionUpdate(subscriptionID string) error
	CreateCustomerPortalSession(userID int, returnURL string) (string, error)
}

type PaymentHandler struct {
	paymentService paymentServicer
	baseURL        string
}

// NewPaymentHandler creates a PaymentHandler. redirectURL is the configured
// GOOGLE_REDIRECT_URL; the scheme+host is extracted as the app's base URL for
// Stripe success/cancel/return URLs, preventing Host-header spoofing.
func NewPaymentHandler(paymentService *services.PaymentService, redirectURL string) *PaymentHandler {
	base := ""
	if u, err := url.Parse(redirectURL); err == nil {
		base = u.Scheme + "://" + u.Host
	}
	return &PaymentHandler{
		paymentService: paymentService,
		baseURL:        base,
	}
}

// CreateCheckoutSession creates a Stripe checkout session
func (ph *PaymentHandler) CreateCheckoutSession(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	req := services.CheckoutSessionRequest{
		UserID:     user.ID,
		SuccessURL: ph.baseURL + "/subscription/success?session_id={CHECKOUT_SESSION_ID}",
		CancelURL:  ph.baseURL + "/subscription/cancel",
	}

	session, err := ph.paymentService.CreateCheckoutSession(req)
	if err != nil {
		if errors.Is(err, services.ErrAlreadySubscribed) {
			c.JSON(http.StatusConflict, gin.H{"error": "You already have an active subscription."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create a checkout session. Please try again."})
		return
	}

	c.JSON(http.StatusOK, session)
}

// GetStripeConfig returns Stripe configuration for frontend
func (ph *PaymentHandler) GetStripeConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"publishable_key": ph.paymentService.GetStripePublishableKey(),
	})
}

// WebhookHandler handles Stripe webhooks
func (ph *PaymentHandler) WebhookHandler(c *gin.Context) {
	const MaxBodyBytes = int64(65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		fmt.Printf("ERROR: Webhook - Failed to read request body: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "The request body could not be read."})
		return
	}

	// Verify webhook signature
	endpointSecret := ph.paymentService.GetStripeWebhookSecret()
	if endpointSecret == "" {
		fmt.Printf("ERROR: Webhook - STRIPE_WEBHOOK_SECRET not configured\n")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "The webhook endpoint is not properly configured.",
		})
		return
	}

	event, err := webhook.ConstructEventWithOptions(payload, c.Request.Header.Get("Stripe-Signature"), endpointSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		fmt.Printf("ERROR: Webhook - Signature verification failed: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook signature verification failed."})
		return
	}

	fmt.Printf("INFO: Webhook - Received event: %s (ID: %s)\n", event.Type, event.ID)

	// Handle the event
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "The webhook payload could not be parsed."})
			return
		}

		// Handle successful checkout
		if session.Mode == stripe.CheckoutSessionModeSubscription {
			// The subscription itself is recorded by the
			// customer.subscription.created event.
			fmt.Printf("INFO: Webhook - Checkout session %s completed\n", redactStripeID(session.ID))
		}

	case "customer.subscription.created", "customer.subscription.updated":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			fmt.Printf("ERROR: Webhook - Failed to parse subscription JSON: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "The webhook payload could not be parsed."})
			return
		}

		redactedSubID := redactStripeID(subscription.ID)

		// Update subscription status in database
		err = ph.paymentService.HandleSubscriptionUpdate(subscription.ID)
		if err != nil {
			fmt.Printf("ERROR: Webhook - Failed to update subscription %s: %v\n", redactedSubID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update the subscription record."})
			return
		}

		fmt.Printf("INFO: Webhook - Updated subscription %s (status: %s)\n", redactedSubID, subscription.Status)

	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "The webhook payload could not be parsed."})
			return
		}

		redactedSubID := redactStripeID(subscription.ID)

		// Handle subscription cancellation
		err = ph.paymentService.HandleSubscriptionUpdate(subscription.ID)
		if err != nil {
			fmt.Printf("ERROR: Webhook - Failed to handle deletion of subscription %s: %v\n", redactedSubID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process the subscription deletion."})
			return
		}

		fmt.Printf("INFO: Webhook - Handled deletion of subscription %s\n", redactedSubID)

	default:
		fmt.Printf("INFO: Webhook - Unhandled event type: %s\n", event.Type)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// CreateCustomerPortal creates a customer portal session for managing subscription
func (ph *PaymentHandler) CreateCustomerPortal(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	returnURL := ph.baseURL + "/subscription"

	portalURL, err := ph.paymentService.CreateCustomerPortalSession(user.ID, returnURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create a customer portal session. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"portal_url": portalURL,
	})
}

// SubscriptionSuccess handles successful subscription redirect
func (ph *PaymentHandler) SubscriptionSuccess(c *gin.Context) {
	sessionID := c.Query("session_id")

	c.HTML(http.StatusOK, "subscription_success.html", gin.H{
		"title":      "Subscription Successful - GoRead2",
		"session_id": sessionID,
	})
}

// SubscriptionCancel handles cancelled subscription redirect
func (ph *PaymentHandler) SubscriptionCancel(c *gin.Context) {
	c.HTML(http.StatusOK, "subscription_cancel.html", gin.H{
		"title": "Subscription Cancelled - GoRead2",
	})
}

// redactStripeID keeps only the first 8 characters of a Stripe object ID,
// enough to tell the object type and correlate log lines without recording
// the full identifier.
func redactStripeID(id string) string {
	if len(id) > 8 {
		return id[:8] + "***"
	}
	return "***"
}
