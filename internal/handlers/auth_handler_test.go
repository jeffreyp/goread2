package handlers

import (
	"testing"

	"github.com/jeffreyp/goread2/internal/auth"
)

func TestNewAuthHandler(t *testing.T) {
	// Create mock services
	mockAuthService := &auth.AuthService{}
	mockSessionManager := &auth.SessionManager{}
	mockCSRFManager := auth.NewCSRFManager()

	handler := NewAuthHandler(mockAuthService, mockSessionManager, mockCSRFManager)

	if handler == nil {
		t.Fatal("NewAuthHandler returned nil")
		return
	}

	if handler.authService != mockAuthService {
		t.Error("AuthHandler auth service not set correctly")
	}

	if handler.sessionManager != mockSessionManager {
		t.Error("AuthHandler session manager not set correctly")
	}

	if handler.csrfManager != mockCSRFManager {
		t.Error("AuthHandler CSRF manager not set correctly")
	}
}
