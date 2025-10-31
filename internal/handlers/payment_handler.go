package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
	"goread2/internal/auth"
	"goread2/internal/services"
)

type PaymentHandler struct {
	paymentService *services.PaymentService
}

func NewPaymentHandler(paymentService *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// CreateCheckoutSession creates a Stripe checkout session
func (ph *PaymentHandler) CreateCheckoutSession(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get the base URL for success/cancel URLs
	scheme := "https"
	if c.Request.Header.Get("X-Forwarded-Proto") == "" && c.Request.TLS == nil {
		scheme = "http"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)

	req := services.CheckoutSessionRequest{
		UserID:     user.ID,
		SuccessURL: baseURL + "/subscription/success?session_id={CHECKOUT_SESSION_ID}",
		CancelURL:  baseURL + "/subscription/cancel",
	}

	session, err := ph.paymentService.CreateCheckoutSession(req)
	if err != nil {
		if err.Error() == "user already has an active subscription" {
			c.JSON(http.StatusConflict, gin.H{"error": "You already have an active subscription"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error reading request body"})
		return
	}

	// Verify webhook signature
	endpointSecret := ph.paymentService.GetStripeWebhookSecret()
	if endpointSecret == "" {
		fmt.Printf("ERROR: Webhook - STRIPE_WEBHOOK_SECRET not configured\n")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Webhook endpoint is not properly configured",
		})
		return
	}

	event, err := webhook.ConstructEventWithOptions(payload, c.Request.Header.Get("Stripe-Signature"), endpointSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		fmt.Printf("ERROR: Webhook - Signature verification failed: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Webhook signature verification failed: %v", err)})
		return
	}

	fmt.Printf("INFO: Webhook - Received event: %s (ID: %s)\n", event.Type, event.ID)

	// Handle the event
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing webhook JSON"})
			return
		}

		// Handle successful checkout
		if session.Mode == stripe.CheckoutSessionModeSubscription {
			// Redact session ID for security
			redactedSessionID := "***"
			if len(session.ID) > 8 {
				redactedSessionID = session.ID[:8] + "***"
			}
			// Subscription will be handled by subscription.created event
			fmt.Printf("Checkout session completed for subscription: %s\n", redactedSessionID)
		}

	case "customer.subscription.created", "customer.subscription.updated":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			fmt.Printf("ERROR: Webhook - Failed to parse subscription JSON: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing webhook JSON"})
			return
		}

		// Redact sensitive IDs for security
		redactedSubID := "***"
		if len(subscription.ID) > 8 {
			redactedSubID = subscription.ID[:8] + "***"
		}
		redactedCustomerID := "***"
		if subscription.Customer != nil && len(subscription.Customer.ID) > 8 {
			redactedCustomerID = subscription.Customer.ID[:8] + "***"
		}

		fmt.Printf("INFO: Webhook - Processing subscription %s (status: %s, customer: %s)\n",
			redactedSubID, subscription.Status, redactedCustomerID)

		// Log metadata for debugging
		if userID, exists := subscription.Metadata["user_id"]; exists {
			fmt.Printf("INFO: Webhook - Found user_id in metadata: %s\n", userID)
		} else {
			fmt.Printf("WARNING: Webhook - No user_id found in subscription metadata\n")
		}

		// Update subscription status in database
		err = ph.paymentService.HandleSubscriptionUpdate(subscription.ID)
		if err != nil {
			fmt.Printf("ERROR: Webhook - Failed to update subscription %s: %v\n", redactedSubID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating subscription"})
			return
		}

		fmt.Printf("SUCCESS: Webhook - Updated subscription %s in database\n", redactedSubID)

	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing webhook JSON"})
			return
		}

		// Redact subscription ID for security
		redactedSubID := "***"
		if len(subscription.ID) > 8 {
			redactedSubID = subscription.ID[:8] + "***"
		}

		// Handle subscription cancellation
		err = ph.paymentService.HandleSubscriptionUpdate(subscription.ID)
		if err != nil {
			fmt.Printf("Error handling subscription deletion %s: %v\n", redactedSubID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error handling subscription deletion"})
			return
		}

		fmt.Printf("Successfully handled subscription deletion: %s\n", redactedSubID)

	default:
		fmt.Printf("Unhandled event type: %s\n", event.Type)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// CreateCustomerPortal creates a customer portal session for managing subscription
func (ph *PaymentHandler) CreateCustomerPortal(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get the base URL for return URL
	scheme := "https"
	if c.Request.Header.Get("X-Forwarded-Proto") == "" && c.Request.TLS == nil {
		scheme = "http"
	}
	returnURL := fmt.Sprintf("%s://%s/subscription", scheme, c.Request.Host)

	portalURL, err := ph.paymentService.CreateCustomerPortalSession(user.ID, returnURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
