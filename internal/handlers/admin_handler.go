package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"goread2/internal/auth"
	"goread2/internal/services"
)

type AdminHandler struct {
	subscriptionService *services.SubscriptionService
	auditService        *services.AuditService
}

func NewAdminHandler(subscriptionService *services.SubscriptionService, auditService *services.AuditService) *AdminHandler {
	return &AdminHandler{
		subscriptionService: subscriptionService,
		auditService:        auditService,
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
		// Log failure
		_ = ah.auditService.LogFailure(
			currentUser.ID,
			currentUser.Email,
			map[bool]string{true: "grant_admin", false: "revoke_admin"}[request.IsAdmin],
			user.ID,
			user.Email,
			map[string]interface{}{
				"is_admin":    request.IsAdmin,
				"user_name":   user.Name,
			},
			c.ClientIP(),
			err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set admin status", "details": err.Error()})
		return
	}

	// Log success
	_ = ah.auditService.LogSuccess(
		currentUser.ID,
		currentUser.Email,
		map[bool]string{true: "grant_admin", false: "revoke_admin"}[request.IsAdmin],
		user.ID,
		user.Email,
		map[string]interface{}{
			"is_admin":    request.IsAdmin,
			"user_name":   user.Name,
		},
		c.ClientIP(),
	)

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
		// Log failure
		_ = ah.auditService.LogFailure(
			currentUser.ID,
			currentUser.Email,
			"grant_free_months",
			user.ID,
			user.Email,
			map[string]interface{}{
				"months_granted": request.Months,
				"user_name":      user.Name,
			},
			c.ClientIP(),
			err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant free months", "details": err.Error()})
		return
	}

	// Log success
	_ = ah.auditService.LogSuccess(
		currentUser.ID,
		currentUser.Email,
		"grant_free_months",
		user.ID,
		user.Email,
		map[string]interface{}{
			"months_granted":    request.Months,
			"user_name":         user.Name,
			"total_free_months": user.FreeMonthsRemaining + request.Months,
		},
		c.ClientIP(),
	)

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

// GetAuditLogs handles GET /admin/audit-logs
func (ah *AdminHandler) GetAuditLogs(c *gin.Context) {
	// Parse query parameters
	limit := 50 // Default limit
	offset := 0 // Default offset

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 100 {
				limit = 100 // Max limit
			}
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Parse filters
	filters := make(map[string]interface{})
	if userID := c.Query("admin_user_id"); userID != "" {
		if id, err := strconv.Atoi(userID); err == nil {
			filters["admin_user_id"] = id
		}
	}
	if targetUserID := c.Query("target_user_id"); targetUserID != "" {
		if id, err := strconv.Atoi(targetUserID); err == nil {
			filters["target_user_id"] = id
		}
	}
	if opType := c.Query("operation_type"); opType != "" {
		filters["operation_type"] = opType
	}

	// Get audit logs
	logs, err := ah.auditService.GetAuditLogs(limit, offset, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit logs", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"limit":  limit,
		"offset": offset,
	})
}

// GetUserInfo handles GET /admin/users/:email
func (ah *AdminHandler) GetUserInfo(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter required"})
		return
	}

	// Get current admin user for audit logging
	currentUser, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
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

	// Log the access
	_ = ah.auditService.LogSuccess(
		currentUser.ID,
		currentUser.Email,
		"view_user_info",
		user.ID,
		user.Email,
		map[string]interface{}{
			"user_name": user.Name,
		},
		c.ClientIP(),
	)

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
