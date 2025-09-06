package handlers

import (
	"testing"

	"goread2/internal/auth"
)

func TestNewAuthHandler(t *testing.T) {
	// Create mock services
	mockAuthService := &auth.AuthService{}
	mockSessionManager := &auth.SessionManager{}

	handler := NewAuthHandler(mockAuthService, mockSessionManager)

	if handler == nil {
		t.Fatal("NewAuthHandler returned nil")
	}

	if handler.authService != mockAuthService {
		t.Error("AuthHandler auth service not set correctly")
	}

	if handler.sessionManager != mockSessionManager {
		t.Error("AuthHandler session manager not set correctly")
	}
}
