package handlers

import (
	"testing"

	"goread2/internal/services"
)

func TestNewAdminHandler(t *testing.T) {
	// Create a mock subscription service
	mockService := &services.SubscriptionService{}

	handler := NewAdminHandler(mockService)

	if handler == nil {
		t.Fatal("NewAdminHandler returned nil")
	}

	if handler.subscriptionService != mockService {
		t.Error("AdminHandler subscription service not set correctly")
	}
}
