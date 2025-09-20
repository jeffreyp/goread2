package services

import (
	"context"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v78"
	billingportalsession "github.com/stripe/stripe-go/v78/billingportal/session"
	"github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/customer"
	"github.com/stripe/stripe-go/v78/price"
	"github.com/stripe/stripe-go/v78/product"
	"github.com/stripe/stripe-go/v78/subscription"
	"goread2/internal/database"
	"goread2/internal/secrets"
)

type PaymentService struct {
	db                  database.Database
	subscriptionService *SubscriptionService
	stripeSecretKey     string
	stripePublishableKey string
	stripeWebhookSecret string
	stripePriceID       string
}

type CheckoutSessionRequest struct {
	UserID     int    `json:"user_id"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}

type CheckoutSessionResponse struct {
	SessionID  string `json:"session_id"`
	SessionURL string `json:"session_url"`
}

func NewPaymentService(db database.Database, subscriptionService *SubscriptionService) *PaymentService {
	ctx := context.Background()
	
	// Get Stripe credentials from Secret Manager or environment
	secretKey, publishableKey, webhookSecret, priceID, err := secrets.GetStripeCredentials(ctx)
	if err != nil {
		fmt.Printf("ERROR: Failed to get Stripe credentials: %v\n", err)
		// Fallback to empty values - this will cause validation errors later
		secretKey, publishableKey, webhookSecret, priceID = "", "", "", ""
	}

	// Initialize Stripe with secret key
	if secretKey == "" {
		fmt.Printf("ERROR: Stripe secret key is empty\n")
	} else if len(secretKey) < 10 {
		fmt.Printf("ERROR: Stripe secret key appears to be too short: %d characters\n", len(secretKey))
	} else {
		// Log first and last 4 characters for debugging (safe)
		fmt.Printf("INFO: Using Stripe key: %s...%s (%d chars total)\n", 
			secretKey[:4], secretKey[len(secretKey)-4:], len(secretKey))
	}
	stripe.Key = secretKey

	return &PaymentService{
		db:                   db,
		subscriptionService:  subscriptionService,
		stripeSecretKey:      secretKey,
		stripePublishableKey: publishableKey,
		stripeWebhookSecret:  webhookSecret,
		stripePriceID:        priceID,
	}
}

// ValidateStripeConfig validates that all required Stripe credentials are set
func (ps *PaymentService) ValidateStripeConfig() error {
	requiredVars := map[string]string{
		"STRIPE_SECRET_KEY":      ps.stripeSecretKey,
		"STRIPE_PUBLISHABLE_KEY": ps.stripePublishableKey,
		"STRIPE_WEBHOOK_SECRET":  ps.stripeWebhookSecret,
		"STRIPE_PRICE_ID":        ps.stripePriceID,
	}

	for varName, value := range requiredVars {
		if value == "" {
			return fmt.Errorf("required Stripe credential %s is not set", varName)
		}
	}

	return nil
}

// CreateCheckoutSession creates a Stripe Checkout session for subscription
func (ps *PaymentService) CreateCheckoutSession(req CheckoutSessionRequest) (*CheckoutSessionResponse, error) {
	// Get user details
	user, err := ps.db.GetUserByID(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user already has an active paid subscription
	// Note: Admin users should be able to subscribe even though they have unlimited access
	if user.SubscriptionStatus == "active" {
		return nil, fmt.Errorf("user already has an active subscription")
	}

	// Create or get Stripe customer
	customerID, err := ps.getOrCreateStripeCustomer(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create customer: %w", err)
	}

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(ps.stripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(req.SuccessURL),
		CancelURL:  stripe.String(req.CancelURL),
		Metadata: map[string]string{
			"user_id": fmt.Sprintf("%d", req.UserID),
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": fmt.Sprintf("%d", req.UserID),
			},
		},
	}

	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return &CheckoutSessionResponse{
		SessionID:  sess.ID,
		SessionURL: sess.URL,
	}, nil
}

// getOrCreateStripeCustomer gets existing customer or creates a new one
func (ps *PaymentService) getOrCreateStripeCustomer(user *database.User) (string, error) {
	// Try to find existing customer by email
	params := &stripe.CustomerListParams{
		Email: stripe.String(user.Email),
	}
	params.Limit = stripe.Int64(1)

	iter := customer.List(params)
	for iter.Next() {
		return iter.Customer().ID, nil
	}

	if err := iter.Err(); err != nil {
		return "", fmt.Errorf("failed to search for customer: %w", err)
	}

	// Create new customer
	customerParams := &stripe.CustomerParams{
		Email: stripe.String(user.Email),
		Name:  stripe.String(user.Name),
		Metadata: map[string]string{
			"user_id":   fmt.Sprintf("%d", user.ID),
			"google_id": user.GoogleID,
		},
	}

	cust, err := customer.New(customerParams)
	if err != nil {
		return "", fmt.Errorf("failed to create customer: %w", err)
	}

	return cust.ID, nil
}

// HandleSubscriptionUpdate handles subscription status changes from webhooks
func (ps *PaymentService) HandleSubscriptionUpdate(subscriptionID string) error {
	fmt.Printf("DEBUG: HandleSubscriptionUpdate - Processing subscription ID: %s\n", subscriptionID)
	
	// Get subscription from Stripe
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		fmt.Printf("ERROR: HandleSubscriptionUpdate - Failed to get subscription from Stripe: %v\n", err)
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	fmt.Printf("DEBUG: HandleSubscriptionUpdate - Subscription status from Stripe: %s\n", sub.Status)

	// Extract user ID from metadata
	userIDStr, exists := sub.Metadata["user_id"]
	if !exists {
		fmt.Printf("ERROR: HandleSubscriptionUpdate - No user_id in subscription metadata. Available metadata: %+v\n", sub.Metadata)
		return fmt.Errorf("user_id not found in subscription metadata")
	}

	var userID int
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		fmt.Printf("ERROR: HandleSubscriptionUpdate - Invalid user_id format '%s': %v\n", userIDStr, err)
		return fmt.Errorf("invalid user_id in metadata: %w", err)
	}

	fmt.Printf("DEBUG: HandleSubscriptionUpdate - Processing for user ID: %d\n", userID)

	// Convert Stripe status to our status
	var status string
	var lastPaymentDate time.Time
	var nextBillingDate time.Time

	switch sub.Status {
	case stripe.SubscriptionStatusActive:
		status = "active"
		if sub.CurrentPeriodStart > 0 {
			lastPaymentDate = time.Unix(sub.CurrentPeriodStart, 0)
		}
		if sub.CurrentPeriodEnd > 0 {
			nextBillingDate = time.Unix(sub.CurrentPeriodEnd, 0)
		}
	case stripe.SubscriptionStatusCanceled:
		status = "cancelled"
	case stripe.SubscriptionStatusPastDue:
		status = "past_due"
	case stripe.SubscriptionStatusUnpaid:
		status = "unpaid"
	default:
		status = "cancelled"
	}

	fmt.Printf("DEBUG: HandleSubscriptionUpdate - Mapped status: %s, payment date: %v, next billing: %v\n", status, lastPaymentDate, nextBillingDate)

	// Update user subscription in database
	err = ps.subscriptionService.UpdateUserSubscription(userID, status, subscriptionID, lastPaymentDate, nextBillingDate)
	if err != nil {
		fmt.Printf("ERROR: HandleSubscriptionUpdate - Database update failed: %v\n", err)
		return fmt.Errorf("failed to update user subscription: %w", err)
	}

	fmt.Printf("SUCCESS: HandleSubscriptionUpdate - Database updated for user %d with status '%s'\n", userID, status)
	return nil
}

// CreateProductAndPrice creates the GoRead2 Pro product and price in Stripe (one-time setup)
func (ps *PaymentService) CreateProductAndPrice() (*stripe.Price, error) {
	// Create product
	productParams := &stripe.ProductParams{
		Name:        stripe.String("GoRead2 Pro"),
		Description: stripe.String("Unlimited RSS feeds and premium features for GoRead2"),
		Type:        stripe.String("service"),
		Metadata: map[string]string{
			"app": "goread2",
		},
	}

	prod, err := product.New(productParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Create price (monthly subscription)
	priceParams := &stripe.PriceParams{
		Product:    stripe.String(prod.ID),
		UnitAmount: stripe.Int64(299), // $2.99 in cents
		Currency:   stripe.String(string(stripe.CurrencyUSD)),
		Recurring: &stripe.PriceRecurringParams{
			Interval: stripe.String(string(stripe.PriceRecurringIntervalMonth)),
		},
		Metadata: map[string]string{
			"app": "goread2",
		},
	}

	priceObj, err := price.New(priceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create price: %w", err)
	}

	return priceObj, nil
}

// GetStripePublishableKey returns the Stripe publishable key for frontend
func (ps *PaymentService) GetStripePublishableKey() string {
	return ps.stripePublishableKey
}

// GetStripeWebhookSecret returns the Stripe webhook secret for verification
func (ps *PaymentService) GetStripeWebhookSecret() string {
	return ps.stripeWebhookSecret
}

// CreateCustomerPortalSession creates a session for customer to manage their subscription
func (ps *PaymentService) CreateCustomerPortalSession(userID int, returnURL string) (string, error) {
	user, err := ps.db.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	// Find Stripe customer
	customerID, err := ps.getOrCreateStripeCustomer(user)
	if err != nil {
		return "", fmt.Errorf("failed to get customer: %w", err)
	}

	// Create billing portal session
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	portalSession, err := billingportalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create customer portal session: %w", err)
	}

	return portalSession.URL, nil
}
