package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error reading request body"})
		return
	}

	// Verify webhook signature
	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	event, err := webhook.ConstructEvent(payload, c.Request.Header.Get("Stripe-Signature"), endpointSecret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Webhook signature verification failed: %v", err)})
		return
	}

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
			// Subscription will be handled by subscription.created event
			fmt.Printf("Checkout session completed for subscription: %s\n", session.ID)
		}

	case "customer.subscription.created", "customer.subscription.updated":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing webhook JSON"})
			return
		}

		// Update subscription status in database
		err = ph.paymentService.HandleSubscriptionUpdate(subscription.ID)
		if err != nil {
			fmt.Printf("Error updating subscription %s: %v\n", subscription.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating subscription"})
			return
		}

		fmt.Printf("Successfully updated subscription: %s\n", subscription.ID)

	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing webhook JSON"})
			return
		}

		// Handle subscription cancellation
		err = ph.paymentService.HandleSubscriptionUpdate(subscription.ID)
		if err != nil {
			fmt.Printf("Error handling subscription deletion %s: %v\n", subscription.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error handling subscription deletion"})
			return
		}

		fmt.Printf("Successfully handled subscription deletion: %s\n", subscription.ID)

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
		"url": portalURL,
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
