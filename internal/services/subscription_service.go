package services

import (
	"errors"
	"log"
	"time"

	"goread2/internal/config"
	"goread2/internal/database"
)

const (
	FreeTrialFeedLimit = 20
)

var (
	ErrFeedLimitReached = errors.New("feed limit reached for trial users")
	ErrTrialExpired     = errors.New("trial period has expired")
)

type SubscriptionService struct {
	db database.Database
}

func NewSubscriptionService(db database.Database) *SubscriptionService {
	return &SubscriptionService{db: db}
}

// CanUserAddFeed checks if a user can add another feed based on their subscription status
func (ss *SubscriptionService) CanUserAddFeed(userID int) error {
	// If subscription system is disabled, allow unlimited feeds for everyone
	if !config.IsSubscriptionEnabled() {
		return nil
	}

	// Get user details first
	user, err := ss.db.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Admin users can add unlimited feeds
	if user.IsAdmin {
		return nil
	}

	// Users with active paid subscription can add unlimited feeds
	if user.SubscriptionStatus == "active" {
		return nil
	}

	// Check for remaining free months
	if user.FreeMonthsRemaining > 0 {
		return nil
	}

	// Check if trial has expired
	if user.SubscriptionStatus == "trial" && time.Now().After(user.TrialEndsAt) {
		return ErrTrialExpired
	}

	// If user is on trial and not expired, check feed limit
	if user.SubscriptionStatus == "trial" {
		currentFeedCount, err := ss.db.GetUserFeedCount(userID)
		if err != nil {
			return err
		}

		if currentFeedCount >= FreeTrialFeedLimit {
			return ErrFeedLimitReached
		}
	}

	return nil
}

// GetUserSubscriptionInfo returns subscription information for the user
func (ss *SubscriptionService) GetUserSubscriptionInfo(userID int) (*SubscriptionInfo, error) {
	// Debug logging to identify production issue
	log.Printf("DEBUG: GetUserSubscriptionInfo called for user %d", userID)
	log.Printf("DEBUG: Subscription enabled: %v", config.IsSubscriptionEnabled())
	
	user, err := ss.db.GetUserByID(userID)
	if err != nil {
		log.Printf("DEBUG: Failed to get user %d: %v", userID, err)
		return nil, err
	}
	log.Printf("DEBUG: User %d - Status: %s, TrialEndsAt: %v, IsAdmin: %v", userID, user.SubscriptionStatus, user.TrialEndsAt, user.IsAdmin)

	feedCount, err := ss.db.GetUserFeedCount(userID)
	if err != nil {
		log.Printf("DEBUG: Failed to get feed count for user %d: %v", userID, err)
		return nil, err
	}
	log.Printf("DEBUG: User %d feed count: %d", userID, feedCount)

	isActive, err := ss.db.IsUserSubscriptionActive(userID)
	if err != nil {
		log.Printf("DEBUG: Failed to check if user %d is active: %v", userID, err)
		return nil, err
	}
	log.Printf("DEBUG: User %d is active: %v", userID, isActive)

	info := &SubscriptionInfo{
		Status:          user.SubscriptionStatus,
		SubscriptionID:  user.SubscriptionID,
		TrialEndsAt:     user.TrialEndsAt,
		LastPaymentDate: user.LastPaymentDate,
		CurrentFeeds:    feedCount,
		IsActive:        isActive,
	}

	// If subscription system is disabled, return unlimited access for everyone
	if !config.IsSubscriptionEnabled() {
		log.Printf("DEBUG: Subscriptions disabled, returning unlimited status for user %d", userID)
		info.Status = "unlimited"
		info.FeedLimit = -1 // Unlimited
		info.CanAddFeeds = true
		return info, nil
	}
	
	log.Printf("DEBUG: Subscriptions enabled, processing subscription logic for user %d", userID)

	// Set feed limit and status based on user type
	if user.IsAdmin {
		info.Status = "admin"
		info.FeedLimit = -1 // Unlimited
		info.CanAddFeeds = true
	} else if user.SubscriptionStatus == "active" || user.FreeMonthsRemaining > 0 {
		if user.FreeMonthsRemaining > 0 {
			info.Status = "free_months"
		}
		info.FeedLimit = -1 // Unlimited
		info.CanAddFeeds = true
	} else {
		info.FeedLimit = FreeTrialFeedLimit
		info.CanAddFeeds = feedCount < FreeTrialFeedLimit && isActive
	}

	// Calculate days remaining in trial
	if user.SubscriptionStatus == "trial" {
		daysRemaining := int(time.Until(user.TrialEndsAt).Hours() / 24)
		if daysRemaining < 0 {
			daysRemaining = 0
		}
		info.TrialDaysRemaining = daysRemaining
	}

	return info, nil
}

// UpdateUserSubscription updates a user's subscription status
func (ss *SubscriptionService) UpdateUserSubscription(userID int, status, subscriptionID string, lastPaymentDate time.Time) error {
	return ss.db.UpdateUserSubscription(userID, status, subscriptionID, lastPaymentDate)
}

// Admin management methods
func (ss *SubscriptionService) SetUserAdmin(userID int, isAdmin bool) error {
	return ss.db.SetUserAdmin(userID, isAdmin)
}

func (ss *SubscriptionService) GrantFreeMonths(userID int, months int) error {
	return ss.db.GrantFreeMonths(userID, months)
}

func (ss *SubscriptionService) GetUserByEmail(email string) (*database.User, error) {
	return ss.db.GetUserByEmail(email)
}

// SubscriptionInfo contains all subscription-related information for a user
type SubscriptionInfo struct {
	Status               string    `json:"status"`
	SubscriptionID       string    `json:"subscription_id"`
	TrialEndsAt          time.Time `json:"trial_ends_at"`
	LastPaymentDate      time.Time `json:"last_payment_date"`
	CurrentFeeds         int       `json:"current_feeds"`
	FeedLimit            int       `json:"feed_limit"` // -1 for unlimited
	CanAddFeeds          bool      `json:"can_add_feeds"`
	IsActive             bool      `json:"is_active"`
	TrialDaysRemaining   int       `json:"trial_days_remaining,omitempty"`
}