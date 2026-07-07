package security

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/jeffreyp/goread2/internal/config"
	"github.com/jeffreyp/goread2/internal/services"
	"github.com/jeffreyp/goread2/test/helpers"
)

// TestAddFeedRejectsTrialUserOverLimit exercises FreeTrialFeedLimit enforcement
// through the real POST /api/feeds handler (see
// internal/services/subscription_service_test.go for the unit-level
// CanUserAddFeed cases), so a regression that only breaks the handler's
// wiring of the check — not the check itself — is still caught.
func TestAddFeedRejectsTrialUserOverLimit(t *testing.T) {
	t.Setenv("SUBSCRIPTION_ENABLED", "true")
	config.ResetForTesting()
	config.Load()
	t.Cleanup(config.ResetForTesting)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)
	user := helpers.CreateTestUser(t, testServer.DB, "feedlimit_google1", "feedlimit_test@example.com", "Feed Limit Test User")

	for i := 0; i < services.FreeTrialFeedLimit; i++ {
		feed := helpers.CreateTestFeed(t, testServer.DB, "Feed "+strconv.Itoa(i), "http://feed"+strconv.Itoa(i)+".example.com/rss", "")
		if err := testServer.DB.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
			t.Fatalf("failed to subscribe user to feed %d: %v", i, err)
		}
	}

	req := testServer.CreateAuthenticatedRequest(t, "POST", "/api/feeds", map[string]string{"url": "http://one-more.example.com/rss"}, user)
	rr := testServer.ExecuteRequest(req)

	if rr.Code != http.StatusPaymentRequired {
		t.Errorf("expected 402 once a trial user is at the %d-feed limit, got %d: %s", services.FreeTrialFeedLimit, rr.Code, rr.Body.String())
	}
}
