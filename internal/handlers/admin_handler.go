package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"goread2/internal/auth"
	"goread2/internal/services"
)

type AdminHandler struct {
	subscriptionService *services.SubscriptionService
}

func NewAdminHandler(subscriptionService *services.SubscriptionService) *AdminHandler {
	return &AdminHandler{
		subscriptionService: subscriptionService,
	}
}

// ListUsers handles GET /admin/users
func (ah *AdminHandler) ListUsers(c *gin.Context) {
	// This would require a more complex implementation to return users as JSON
	// For now, return a simple response indicating this needs to be implemented
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "List users API not yet implemented",
		"note":  "Use the CLI tool for now with proper ADMIN_TOKEN authentication",
	})
}

// SetAdminStatus handles POST /admin/users/:email/admin
func (ah *AdminHandler) SetAdminStatus(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter required"})
		return
	}

	var request struct {
		IsAdmin bool `json:"is_admin" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get current admin user for audit logging
	currentUser, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Find target user
	user, err := ah.subscriptionService.GetUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found", "details": err.Error()})
		return
	}

	// Prevent self-demotion (admin removing their own admin status)
	if user.ID == currentUser.ID && !request.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot remove your own admin privileges"})
		return
	}

	// Set admin status
	err = ah.subscriptionService.SetUserAdmin(user.ID, request.IsAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set admin status", "details": err.Error()})
		return
	}

	// TODO: Add audit logging here
	// log.Printf("Admin %s (%d) %s admin privileges for user %s (%d)",
	//     currentUser.Email, currentUser.ID,
	//     map[bool]string{true: "granted", false: "removed"}[request.IsAdmin],
	//     user.Email, user.ID)

	status := "removed from"
	if request.IsAdmin {
		status = "granted to"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Admin status updated successfully",
		"details": map[string]interface{}{
			"user_email":   user.Email,
			"user_name":    user.Name,
			"is_admin":     request.IsAdmin,
			"action":       "Admin privileges " + status + " user",
			"performed_by": currentUser.Email,
		},
	})
}

// GrantFreeMonths handles POST /admin/users/:email/free-months
func (ah *AdminHandler) GrantFreeMonths(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter required"})
		return
	}

	var request struct {
		Months int `json:"months" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get current admin user for audit logging
	currentUser, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Find target user
	user, err := ah.subscriptionService.GetUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found", "details": err.Error()})
		return
	}

	// Grant free months
	err = ah.subscriptionService.GrantFreeMonths(user.ID, request.Months)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant free months", "details": err.Error()})
		return
	}

	// TODO: Add audit logging here
	// log.Printf("Admin %s (%d) granted %d free months to user %s (%d)",
	//     currentUser.Email, currentUser.ID, request.Months, user.Email, user.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Free months granted successfully",
		"details": map[string]interface{}{
			"user_email":        user.Email,
			"user_name":         user.Name,
			"months_granted":    request.Months,
			"total_free_months": user.FreeMonthsRemaining + request.Months,
			"performed_by":      currentUser.Email,
		},
	})
}

// GetUserInfo handles GET /admin/users/:email
func (ah *AdminHandler) GetUserInfo(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter required"})
		return
	}

	// Find user
	user, err := ah.subscriptionService.GetUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found", "details": err.Error()})
		return
	}

	// Get subscription info
	subscriptionInfo, err := ah.subscriptionService.GetUserSubscriptionInfo(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subscription info", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":                    user.ID,
			"email":                 user.Email,
			"name":                  user.Name,
			"google_id":             user.GoogleID,
			"is_admin":              user.IsAdmin,
			"subscription_status":   user.SubscriptionStatus,
			"subscription_id":       user.SubscriptionID,
			"free_months_remaining": user.FreeMonthsRemaining,
			"trial_ends_at":         user.TrialEndsAt,
			"last_payment_date":     user.LastPaymentDate,
			"created_at":            user.CreatedAt,
		},
		"subscription_info": subscriptionInfo,
	})
}
