package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
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

// GetDB returns the database instance (for admin commands)
func (ss *SubscriptionService) GetDB() database.Database {
	return ss.db
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

	// Check if trial has expired (only for users actually on trial status)
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
	user, err := ss.db.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	feedCount, err := ss.db.GetUserFeedCount(userID)
	if err != nil {
		return nil, err
	}

	isActive, err := ss.db.IsUserSubscriptionActive(userID)
	if err != nil {
		return nil, err
	}

	info := &SubscriptionInfo{
		Status:          user.SubscriptionStatus,
		SubscriptionID:  user.SubscriptionID,
		TrialEndsAt:     user.TrialEndsAt,
		LastPaymentDate: user.LastPaymentDate,
		NextBillingDate: user.NextBillingDate,
		CurrentFeeds:    feedCount,
		IsActive:        isActive,
	}

	// If subscription system is disabled, return unlimited access for everyone
	if !config.IsSubscriptionEnabled() {
		info.Status = "unlimited"
		info.FeedLimit = -1 // Unlimited
		info.CanAddFeeds = true
		return info, nil
	}

	// Set feed limit and status based on user type
	if user.IsAdmin && user.SubscriptionStatus == "active" {
		// Admin with active subscription
		info.Status = "admin"
		info.FeedLimit = -1 // Unlimited
		info.CanAddFeeds = true
	} else if user.IsAdmin {
		// Admin without subscription - use admin_trial to differentiate
		info.Status = "admin_trial"
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
func (ss *SubscriptionService) UpdateUserSubscription(userID int, status, subscriptionID string, lastPaymentDate, nextBillingDate time.Time) error {
	return ss.db.UpdateUserSubscription(userID, status, subscriptionID, lastPaymentDate, nextBillingDate)
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
	Status             string    `json:"status"`
	SubscriptionID     string    `json:"subscription_id"`
	TrialEndsAt        time.Time `json:"trial_ends_at"`
	LastPaymentDate    time.Time `json:"last_payment_date"`
	NextBillingDate    time.Time `json:"next_billing_date"`
	CurrentFeeds       int       `json:"current_feeds"`
	FeedLimit          int       `json:"feed_limit"` // -1 for unlimited
	CanAddFeeds        bool      `json:"can_add_feeds"`
	IsActive           bool      `json:"is_active"`
	TrialDaysRemaining int       `json:"trial_days_remaining,omitempty"`
}

// AdminToken represents a secure admin authentication token
type AdminToken struct {
	ID          int       `db:"id" json:"id"`
	TokenHash   string    `db:"token_hash" json:"-"` // Never expose in JSON
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	LastUsedAt  time.Time `db:"last_used_at" json:"last_used_at"`
	Description string    `db:"description" json:"description"`
	IsActive    bool      `db:"is_active" json:"is_active"`
}

// GenerateAdminToken creates a new cryptographically secure admin token
func (ss *SubscriptionService) GenerateAdminToken(description string) (string, error) {
	// Generate 32 bytes of random data for the token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	// Convert to hex string (64 characters)
	token := hex.EncodeToString(tokenBytes)

	// Hash the token for database storage
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])
	now := time.Now()

	// Store the hashed token in the database
	if sqliteDB, ok := ss.db.(*database.DB); ok {
		query := `INSERT INTO admin_tokens (token_hash, description, created_at, last_used_at, is_active) 
				  VALUES (?, ?, ?, ?, ?)`
		_, err := sqliteDB.Exec(query, tokenHash, description, now, now, true)
		if err != nil {
			return "", fmt.Errorf("failed to store admin token: %w", err)
		}
	} else if datastoreDB, ok := ss.db.(*database.DatastoreDB); ok {
		ctx := context.Background()
		entity := &database.AdminTokenEntity{
			TokenHash:   tokenHash,
			Description: description,
			CreatedAt:   now,
			LastUsedAt:  now,
			IsActive:    true,
		}

		key := datastore.IncompleteKey("AdminToken", nil)
		_, err := datastoreDB.GetClient().Put(ctx, key, entity)
		if err != nil {
			return "", fmt.Errorf("failed to store admin token in datastore: %w", err)
		}
	} else {
		return "", errors.New("admin token storage not supported for this database type")
	}

	// Return the plain token (this is the only time it's visible)
	return token, nil
}

// ValidateAdminToken checks if a token is valid and updates last_used_at
func (ss *SubscriptionService) ValidateAdminToken(token string) (bool, error) {
	if len(token) != 64 { // 32 bytes as hex = 64 characters
		return false, nil
	}

	// Hash the provided token
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	// Check if token exists and is active
	if sqliteDB, ok := ss.db.(*database.DB); ok {
		var adminToken AdminToken
		query := `SELECT id, token_hash, created_at, last_used_at, description, is_active 
				  FROM admin_tokens WHERE token_hash = ? AND is_active = 1`

		err := sqliteDB.QueryRow(query, tokenHash).Scan(
			&adminToken.ID,
			&adminToken.TokenHash,
			&adminToken.CreatedAt,
			&adminToken.LastUsedAt,
			&adminToken.Description,
			&adminToken.IsActive,
		)

		if err != nil {
			return false, nil // Token not found or invalid
		}

		// Update last used timestamp
		updateQuery := `UPDATE admin_tokens SET last_used_at = ? WHERE id = ?`
		_, err = sqliteDB.Exec(updateQuery, time.Now(), adminToken.ID)
		if err != nil {
			// Log error but don't fail validation
			fmt.Printf("Warning: Failed to update last_used_at for admin token: %v\n", err)
		}

		return true, nil
	} else if datastoreDB, ok := ss.db.(*database.DatastoreDB); ok {
		ctx := context.Background()

		// Query for active token with matching hash
		query := datastore.NewQuery("AdminToken").
			FilterField("token_hash", "=", tokenHash).
			FilterField("is_active", "=", true).
			Limit(1)

		var entities []*database.AdminTokenEntity
		keys, err := datastoreDB.GetClient().GetAll(ctx, query, &entities)
		if err != nil {
			return false, fmt.Errorf("failed to query admin token: %w", err)
		}

		if len(entities) == 0 {
			return false, nil // Token not found or inactive
		}

		// Update last used timestamp
		entity := entities[0]
		entity.LastUsedAt = time.Now()

		_, err = datastoreDB.GetClient().Put(ctx, keys[0], entity)
		if err != nil {
			// Log error but don't fail validation
			fmt.Printf("Warning: Failed to update last_used_at for admin token: %v\n", err)
		}

		return true, nil
	}

	return false, errors.New("admin token validation not supported for this database type")
}

// ListAdminTokens returns all admin tokens (without the actual token values)
func (ss *SubscriptionService) ListAdminTokens() ([]AdminToken, error) {
	if sqliteDB, ok := ss.db.(*database.DB); ok {
		query := `SELECT id, token_hash, created_at, last_used_at, description, is_active 
				  FROM admin_tokens ORDER BY created_at DESC`

		rows, err := sqliteDB.Query(query)
		if err != nil {
			return nil, fmt.Errorf("failed to query admin tokens: %w", err)
		}
		defer func() { _ = rows.Close() }()

		var tokens []AdminToken
		for rows.Next() {
			var token AdminToken
			err := rows.Scan(
				&token.ID,
				&token.TokenHash,
				&token.CreatedAt,
				&token.LastUsedAt,
				&token.Description,
				&token.IsActive,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan admin token: %w", err)
			}
			tokens = append(tokens, token)
		}

		return tokens, nil
	} else if datastoreDB, ok := ss.db.(*database.DatastoreDB); ok {
		ctx := context.Background()

		query := datastore.NewQuery("AdminToken").Order("-created_at")
		var entities []*database.AdminTokenEntity
		keys, err := datastoreDB.GetClient().GetAll(ctx, query, &entities)
		if err != nil {
			return nil, fmt.Errorf("failed to query admin tokens: %w", err)
		}

		var tokens []AdminToken
		for i, entity := range entities {
			tokens = append(tokens, AdminToken{
				ID:          int(keys[i].ID),
				TokenHash:   entity.TokenHash,
				CreatedAt:   entity.CreatedAt,
				LastUsedAt:  entity.LastUsedAt,
				Description: entity.Description,
				IsActive:    entity.IsActive,
			})
		}

		return tokens, nil
	}

	return nil, errors.New("admin token listing not supported for this database type")
}

// RevokeAdminToken deactivates an admin token
func (ss *SubscriptionService) RevokeAdminToken(tokenID int) error {
	if sqliteDB, ok := ss.db.(*database.DB); ok {
		query := `UPDATE admin_tokens SET is_active = 0 WHERE id = ? AND is_active = 1`
		result, err := sqliteDB.Exec(query, tokenID)
		if err != nil {
			return fmt.Errorf("failed to revoke admin token: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to check revoke result: %w", err)
		}

		if rowsAffected == 0 {
			return errors.New("admin token not found")
		}

		return nil
	} else if datastoreDB, ok := ss.db.(*database.DatastoreDB); ok {
		ctx := context.Background()

		// Get the token by ID
		key := datastore.IDKey("AdminToken", int64(tokenID), nil)
		var entity database.AdminTokenEntity

		err := datastoreDB.GetClient().Get(ctx, key, &entity)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				return errors.New("admin token not found")
			}
			return fmt.Errorf("failed to get admin token: %w", err)
		}

		// Check if already inactive
		if !entity.IsActive {
			return errors.New("admin token not found")
		}

		// Set as inactive
		entity.IsActive = false

		_, err = datastoreDB.GetClient().Put(ctx, key, &entity)
		if err != nil {
			return fmt.Errorf("failed to revoke admin token: %w", err)
		}

		return nil
	}

	return errors.New("admin token revocation not supported for this database type")
}

// HasAdminTokens checks if any active admin tokens exist
func (ss *SubscriptionService) HasAdminTokens() (bool, error) {
	if sqliteDB, ok := ss.db.(*database.DB); ok {
		var count int
		query := `SELECT COUNT(*) FROM admin_tokens WHERE is_active = 1`
		err := sqliteDB.QueryRow(query).Scan(&count)
		if err != nil {
			return false, fmt.Errorf("failed to count admin tokens: %w", err)
		}
		return count > 0, nil
	} else if datastoreDB, ok := ss.db.(*database.DatastoreDB); ok {
		ctx := context.Background()

		query := datastore.NewQuery("AdminToken").
			FilterField("is_active", "=", true).
			Limit(1)

		count, err := datastoreDB.GetClient().Count(ctx, query)
		if err != nil {
			return false, fmt.Errorf("failed to count admin tokens: %w", err)
		}

		return count > 0, nil
	}

	return false, errors.New("admin token check not supported for this database type")
}
