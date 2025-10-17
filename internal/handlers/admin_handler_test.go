package handlers

import (
	"testing"

	"goread2/internal/services"
)

func TestNewAdminHandler(t *testing.T) {
	// Create mock services
	mockSubscriptionService := &services.SubscriptionService{}
	mockAuditService := &services.AuditService{}

	handler := NewAdminHandler(mockSubscriptionService, mockAuditService)

	if handler == nil {
		t.Fatal("NewAdminHandler returned nil")
	}

	if handler.subscriptionService != mockSubscriptionService {
		t.Error("AdminHandler subscription service not set correctly")
	}

	if handler.auditService != mockAuditService {
		t.Error("AdminHandler audit service not set correctly")
	}
}
