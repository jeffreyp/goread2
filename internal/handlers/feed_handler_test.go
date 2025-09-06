package handlers

import (
	"testing"

	"goread2/internal/services"
)

func TestNewFeedHandler(t *testing.T) {
	// Create mock services
	mockFeedService := &services.FeedService{}
	mockSubscriptionService := &services.SubscriptionService{}
	
	handler := NewFeedHandler(mockFeedService, mockSubscriptionService)
	
	if handler == nil {
		t.Fatal("NewFeedHandler returned nil")
	}
	
	if handler.feedService != mockFeedService {
		t.Error("FeedHandler feed service not set correctly")
	}
	
	if handler.subscriptionService != mockSubscriptionService {
		t.Error("FeedHandler subscription service not set correctly")
	}
}