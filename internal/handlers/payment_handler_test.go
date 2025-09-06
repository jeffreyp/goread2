package handlers

import (
	"testing"

	"goread2/internal/services"
)

func TestNewPaymentHandler(t *testing.T) {
	// Create mock service
	mockPaymentService := &services.PaymentService{}

	handler := NewPaymentHandler(mockPaymentService)

	if handler == nil {
		t.Fatal("NewPaymentHandler returned nil")
	}

	if handler.paymentService != mockPaymentService {
		t.Error("PaymentHandler payment service not set correctly")
	}
}
