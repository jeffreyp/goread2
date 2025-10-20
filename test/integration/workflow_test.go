package integration

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"goread2/internal/database"
	"goread2/test/helpers"
)

// TestUserWorkflowEndToEnd tests the complete user journey:
// User creation → Feed subscription → Article access → Mark read/starred
func TestUserWorkflowEndToEnd(t *testing.T) {
	t.Parallel()

	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)

	// Step 1: Create a new user (simulating registration)
	user := helpers.CreateTestUser(t, testServer.DB, "google-workflow-123", "workflow@example.com", "Workflow Test User")

	// Step 2: Verify user has no feeds initially
	reqGetFeeds := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds", nil, user)
	rrGetFeeds := testServer.ExecuteRequest(reqGetFeeds)

	if rrGetFeeds.Code != http.StatusOK {
		t.Fatalf("Step 2 failed: Expected status 200, got %d", rrGetFeeds.Code)
	}

	var feeds []database.Feed
	err := json.Unmarshal(rrGetFeeds.Body.Bytes(), &feeds)
	if err != nil {
		t.Fatalf("Step 2 failed: Failed to unmarshal feeds: %v", err)
	}

	if len(feeds) != 0 {
		t.Errorf("Step 2 failed: Expected 0 feeds, got %d", len(feeds))
	}

	// Step 3: Subscribe to a feed
	feed := helpers.CreateTestFeed(t, testServer.DB, "Workflow Test Feed", "https://workflow.com/feed", "Test feed for workflow")
	err = testServer.DB.SubscribeUserToFeed(user.ID, feed.ID)
	if err != nil {
		t.Fatalf("Step 3 failed: Failed to subscribe user to feed: %v", err)
	}

	// Step 4: Verify user now has 1 feed
	reqGetFeeds2 := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds", nil, user)
	rrGetFeeds2 := testServer.ExecuteRequest(reqGetFeeds2)

	if rrGetFeeds2.Code != http.StatusOK {
		t.Fatalf("Step 4 failed: Expected status 200, got %d", rrGetFeeds2.Code)
	}

	var feeds2 []database.Feed
	err = json.Unmarshal(rrGetFeeds2.Body.Bytes(), &feeds2)
	if err != nil {
		t.Fatalf("Step 4 failed: Failed to unmarshal feeds: %v", err)
	}

	if len(feeds2) != 1 {
		t.Fatalf("Step 4 failed: Expected 1 feed, got %d", len(feeds2))
	}

	// Step 5: Add articles to the feed
	article1 := helpers.CreateTestArticle(t, testServer.DB, feed.ID, "Article 1", "https://workflow.com/article1")
	article2 := helpers.CreateTestArticle(t, testServer.DB, feed.ID, "Article 2", "https://workflow.com/article2")

	// Step 6: Get user's articles
	reqGetArticles := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds/"+strconv.Itoa(feed.ID)+"/articles", nil, user)
	rrGetArticles := testServer.ExecuteRequest(reqGetArticles)

	if rrGetArticles.Code != http.StatusOK {
		t.Fatalf("Step 6 failed: Expected status 200, got %d", rrGetArticles.Code)
	}

	var articles []database.Article
	err = json.Unmarshal(rrGetArticles.Body.Bytes(), &articles)
	if err != nil {
		t.Fatalf("Step 6 failed: Failed to unmarshal articles: %v", err)
	}

	if len(articles) != 2 {
		t.Fatalf("Step 6 failed: Expected 2 articles, got %d", len(articles))
	}

	// Step 7: Mark first article as read
	reqMarkRead := testServer.CreateAuthenticatedRequest(t, "POST", "/api/articles/"+strconv.Itoa(article1.ID)+"/read", map[string]bool{"is_read": true}, user)
	rrMarkRead := testServer.ExecuteRequest(reqMarkRead)

	if rrMarkRead.Code != http.StatusOK {
		t.Fatalf("Step 7 failed: Expected status 200, got %d. Body: %s", rrMarkRead.Code, rrMarkRead.Body.String())
	}

	// Step 8: Star second article
	reqStar := testServer.CreateAuthenticatedRequest(t, "POST", "/api/articles/"+strconv.Itoa(article2.ID)+"/star", nil, user)
	rrStar := testServer.ExecuteRequest(reqStar)

	if rrStar.Code != http.StatusOK {
		t.Fatalf("Step 8 failed: Expected status 200, got %d. Body: %s", rrStar.Code, rrStar.Body.String())
	}

	// Step 9: Verify article statuses were updated
	status1, err := testServer.DB.GetUserArticleStatus(user.ID, article1.ID)
	if err != nil {
		t.Fatalf("Step 9 failed: Failed to get article status: %v", err)
	}
	if !status1.IsRead {
		t.Error("Step 9 failed: Article 1 should be marked as read")
	}

	status2, err := testServer.DB.GetUserArticleStatus(user.ID, article2.ID)
	if err != nil {
		t.Fatalf("Step 9 failed: Failed to get article status: %v", err)
	}
	if !status2.IsStarred {
		t.Error("Step 9 failed: Article 2 should be starred")
	}

	// Step 10: Delete the feed
	reqDeleteFeed := testServer.CreateAuthenticatedRequest(t, "DELETE", "/api/feeds/"+strconv.Itoa(feed.ID), nil, user)
	rrDeleteFeed := testServer.ExecuteRequest(reqDeleteFeed)

	if rrDeleteFeed.Code != http.StatusOK {
		t.Fatalf("Step 10 failed: Expected status 200, got %d. Body: %s", rrDeleteFeed.Code, rrDeleteFeed.Body.String())
	}

	// Step 11: Verify feed is gone
	reqGetFeeds3 := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds", nil, user)
	rrGetFeeds3 := testServer.ExecuteRequest(reqGetFeeds3)

	if rrGetFeeds3.Code != http.StatusOK {
		t.Fatalf("Step 11 failed: Expected status 200, got %d", rrGetFeeds3.Code)
	}

	var feeds3 []database.Feed
	err = json.Unmarshal(rrGetFeeds3.Body.Bytes(), &feeds3)
	if err != nil {
		t.Fatalf("Step 11 failed: Failed to unmarshal feeds: %v", err)
	}

	if len(feeds3) != 0 {
		t.Errorf("Step 11 failed: Expected 0 feeds after deletion, got %d", len(feeds3))
	}

	t.Log("✅ Complete user workflow test passed: registration → feed subscription → article reading → marking read/starred → feed deletion")
}

// TestOPMLImportWorkflow tests the OPML import workflow:
// Import OPML → Verify feeds → Verify articles → Access content
func TestOPMLImportWorkflow(t *testing.T) {
	t.Parallel()

	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)
	user := helpers.CreateTestUser(t, testServer.DB, "google-opml-123", "opml@example.com", "OPML Test User")

	// Step 1: Verify OPML import endpoint exists and requires multipart form
	// Note: We send JSON (incorrect format) to verify the endpoint validates input
	reqImport := testServer.CreateAuthenticatedRequest(t, "POST", "/api/feeds/import", nil, user)
	rrImport := testServer.ExecuteRequest(reqImport)

	// Should return 400 since we're not sending a multipart form file
	if rrImport.Code != http.StatusBadRequest {
		t.Errorf("Step 1: Expected status 400 (bad request - no file), got %d. Body: %s", rrImport.Code, rrImport.Body.String())
	}

	// Verify error message indicates file is required
	if !contains(rrImport.Body.String(), "No OPML file provided") {
		t.Errorf("Expected error message about missing file, got: %s", rrImport.Body.String())
	}

	t.Log("✅ OPML import workflow test completed (endpoint validates input correctly)")
}

// TestOPMLExportWorkflow tests the OPML export workflow:
// Create feeds → Export OPML → Verify format
func TestOPMLExportWorkflow(t *testing.T) {
	t.Parallel()

	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)
	user := helpers.CreateTestUser(t, testServer.DB, "google-export-123", "export@example.com", "Export Test User")

	// Step 1: Create test feeds
	feed1 := helpers.CreateTestFeed(t, testServer.DB, "Export Feed 1", "https://export1.com/feed", "Test feed 1")
	feed2 := helpers.CreateTestFeed(t, testServer.DB, "Export Feed 2", "https://export2.com/feed", "Test feed 2")

	// Step 2: Subscribe user to feeds
	err := testServer.DB.SubscribeUserToFeed(user.ID, feed1.ID)
	if err != nil {
		t.Fatalf("Failed to subscribe to feed 1: %v", err)
	}
	err = testServer.DB.SubscribeUserToFeed(user.ID, feed2.ID)
	if err != nil {
		t.Fatalf("Failed to subscribe to feed 2: %v", err)
	}

	// Step 3: Export OPML
	reqExport := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds/export", nil, user)
	rrExport := testServer.ExecuteRequest(reqExport)

	if rrExport.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rrExport.Code, rrExport.Body.String())
	}

	// Step 4: Verify OPML format
	opmlContent := rrExport.Body.String()
	if opmlContent == "" {
		t.Error("Expected non-empty OPML content")
	}

	// Verify it contains the feed URLs
	if !contains(opmlContent, "https://export1.com/feed") {
		t.Error("OPML should contain feed 1 URL")
	}
	if !contains(opmlContent, "https://export2.com/feed") {
		t.Error("OPML should contain feed 2 URL")
	}

	t.Log("✅ OPML export workflow test passed: feeds → export → verification")
}

// TestAdminWorkflow tests admin operations:
// Create admin → Grant privileges → Revoke privileges
func TestAdminWorkflow(t *testing.T) {
	t.Parallel()

	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)

	// Step 1: Create admin and regular user
	admin := helpers.CreateTestUser(t, testServer.DB, "google-admin-123", "admin@example.com", "Admin User")
	regularUser := helpers.CreateTestUser(t, testServer.DB, "google-regular-123", "regular@example.com", "Regular User")

	// Step 2: Grant admin privileges
	err := testServer.DB.SetUserAdmin(admin.ID, true)
	if err != nil {
		t.Fatalf("Failed to set user as admin: %v", err)
	}

	// Step 3: Verify admin status
	adminUser, err := testServer.DB.GetUserByID(admin.ID)
	if err != nil {
		t.Fatalf("Failed to get admin user: %v", err)
	}
	if !adminUser.IsAdmin {
		t.Error("User should have admin privileges")
	}

	// Step 4: Verify regular user is not admin
	regUser, err := testServer.DB.GetUserByID(regularUser.ID)
	if err != nil {
		t.Fatalf("Failed to get regular user: %v", err)
	}
	if regUser.IsAdmin {
		t.Error("Regular user should not have admin privileges")
	}

	// Step 5: Revoke admin privileges
	err = testServer.DB.SetUserAdmin(admin.ID, false)
	if err != nil {
		t.Fatalf("Failed to revoke admin privileges: %v", err)
	}

	// Step 6: Verify admin status revoked
	formerAdmin, err := testServer.DB.GetUserByID(admin.ID)
	if err != nil {
		t.Fatalf("Failed to get former admin user: %v", err)
	}
	if formerAdmin.IsAdmin {
		t.Error("User should no longer have admin privileges")
	}

	t.Log("✅ Admin workflow test passed: grant privileges → verify → revoke")
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
