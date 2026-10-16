package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/secrets"
	"github.com/jeffreyp/goread2/test/helpers"
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

	var articlesResp struct {
		Articles   []database.Article `json:"articles"`
		NextCursor string             `json:"next_cursor"`
	}
	err = json.Unmarshal(rrGetArticles.Body.Bytes(), &articlesResp)
	if err != nil {
		t.Fatalf("Step 6 failed: Failed to unmarshal articles: %v", err)
	}
	articles := articlesResp.Articles

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
	if !contains(rrImport.Body.String(), "No OPML file was included in the request.") {
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

// TestDataExportWorkflow tests the GDPR data export:
// Create feeds, articles, and audit entries → Export → Verify contents and isolation
func TestDataExportWorkflow(t *testing.T) {
	t.Parallel()

	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)
	user := helpers.CreateTestUser(t, testServer.DB, "google-dataexport-123", "dataexport@example.com", "Data Export User")
	other := helpers.CreateTestUser(t, testServer.DB, "google-dataexport-456", "other@example.com", "Other User")

	feed := helpers.CreateTestFeed(t, testServer.DB, "Data Export Feed", "https://dataexport.com/feed", "Test feed")
	if err := testServer.DB.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	read := helpers.CreateTestArticle(t, testServer.DB, feed.ID, "Read Article", "https://dataexport.com/read")
	starred := helpers.CreateTestArticle(t, testServer.DB, feed.ID, "Starred Article", "https://dataexport.com/starred")
	untouched := helpers.CreateTestArticle(t, testServer.DB, feed.ID, "Untouched Article", "https://dataexport.com/untouched")
	if err := testServer.DB.SetUserArticleStatus(user.ID, read.ID, true, false); err != nil {
		t.Fatalf("Failed to mark read: %v", err)
	}
	if err := testServer.DB.SetUserArticleStatus(user.ID, starred.ID, false, true); err != nil {
		t.Fatalf("Failed to star: %v", err)
	}
	// Another user's history must not leak into the export.
	if err := testServer.DB.SetUserArticleStatus(other.ID, untouched.ID, true, true); err != nil {
		t.Fatalf("Failed to set other user's status: %v", err)
	}
	if err := testServer.DB.CreateAuditLog(&database.AuditLog{
		AdminUserID:     other.ID,
		AdminEmail:      other.Email,
		OperationType:   "grant_free_months",
		TargetUserID:    user.ID,
		TargetUserEmail: user.Email,
		IPAddress:       "203.0.113.7",
		Result:          "success",
	}); err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	req := testServer.CreateAuthenticatedRequest(t, "GET", "/api/account/export", nil, user)
	rr := testServer.ExecuteRequest(req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
	if cd := rr.Header().Get("Content-Disposition"); !contains(cd, "attachment; filename=goread2-data-") {
		t.Errorf("Unexpected Content-Disposition: %q", cd)
	}

	body := rr.Body.String()
	for _, leaked := range []string{"203.0.113.7", "other@example.com", "Untouched Article"} {
		if contains(body, leaked) {
			t.Errorf("Export should not contain %q", leaked)
		}
	}

	var export struct {
		Account  database.User `json:"account"`
		Feeds    []struct{ URL string }
		Sessions []map[string]interface{}
		AuditLog []struct {
			Role          string `json:"role"`
			OperationType string `json:"operation_type"`
		} `json:"audit_log"`
		Articles []database.UserArticleRecord
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &export); err != nil {
		t.Fatalf("Export is not valid JSON: %v. Body: %s", err, body)
	}

	if export.Account.Email != user.Email {
		t.Errorf("Expected account email %q, got %q", user.Email, export.Account.Email)
	}
	if len(export.Feeds) != 1 || export.Feeds[0].URL != feed.URL {
		t.Errorf("Expected one feed %q, got %+v", feed.URL, export.Feeds)
	}
	if len(export.Sessions) == 0 {
		t.Error("Expected at least the request's own session")
	}
	for _, session := range export.Sessions {
		if _, ok := session["id"]; ok {
			t.Error("Exported sessions must not include the session ID")
		}
	}
	if len(export.AuditLog) != 1 || export.AuditLog[0].Role != "subject" || export.AuditLog[0].OperationType != "grant_free_months" {
		t.Errorf("Expected one subject audit entry, got %+v", export.AuditLog)
	}

	if len(export.Articles) != 2 {
		t.Fatalf("Expected 2 article records, got %d: %+v", len(export.Articles), export.Articles)
	}
	byID := map[int]database.UserArticleRecord{}
	for _, record := range export.Articles {
		byID[record.ArticleID] = record
	}
	if r := byID[read.ID]; !r.IsRead || r.IsStarred || r.Title != "Read Article" || r.URL != read.URL {
		t.Errorf("Unexpected read article record: %+v", r)
	}
	if r := byID[starred.ID]; r.IsRead || !r.IsStarred || r.Title != "Starred Article" {
		t.Errorf("Unexpected starred article record: %+v", r)
	}
}

// TestAccountDeletionWorkflow tests GDPR account deletion:
// Mismatched confirmation is rejected → Delete → Data, sessions, and cookie are gone,
// another user's data is untouched, and a tombstone records the deleted Google account
func TestAccountDeletionWorkflow(t *testing.T) {
	// Not parallel: it sets ACCOUNT_TOMBSTONE_KEY and resets the process-wide secrets cache.
	t.Setenv("ACCOUNT_TOMBSTONE_KEY", "test-tombstone-key")
	secrets.ResetCacheForTesting()
	t.Cleanup(secrets.ResetCacheForTesting)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)
	user := helpers.CreateTestUser(t, testServer.DB, "google-delete-123", "Delete.Me@example.com", "Delete Test User")
	other := helpers.CreateTestUser(t, testServer.DB, "google-delete-456", "keep@example.com", "Keep Test User")

	feed := helpers.CreateTestFeed(t, testServer.DB, "Deletion Feed", "https://deletion.com/feed", "Test feed")
	article := helpers.CreateTestArticle(t, testServer.DB, feed.ID, "Deletion Article", "https://deletion.com/article")
	for _, u := range []*database.User{user, other} {
		if err := testServer.DB.SubscribeUserToFeed(u.ID, feed.ID); err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}
		if err := testServer.DB.SetUserArticleStatus(u.ID, article.ID, true, true); err != nil {
			t.Fatalf("Failed to set article status: %v", err)
		}
	}

	// Step 1: a confirmation that does not match the account email is rejected
	reqWrong := testServer.CreateAuthenticatedRequest(t, "DELETE", "/api/account", map[string]string{"confirm_email": "someone@example.com"}, user)
	rrWrong := testServer.ExecuteRequest(reqWrong)
	if rrWrong.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for mismatched email, got %d: %s", rrWrong.Code, rrWrong.Body.String())
	}
	if _, err := testServer.DB.GetUserByID(user.ID); err != nil {
		t.Fatalf("User should still exist after a rejected deletion: %v", err)
	}

	// Step 2: delete, confirming with different case and surrounding spaces
	reqDelete := testServer.CreateAuthenticatedRequest(t, "DELETE", "/api/account", map[string]string{"confirm_email": "  delete.me@EXAMPLE.com "}, user)
	rrDelete := testServer.ExecuteRequest(reqDelete)
	if rrDelete.Code != http.StatusOK {
		t.Fatalf("Expected 200 for in-process deletion, got %d: %s", rrDelete.Code, rrDelete.Body.String())
	}
	if setCookie := rrDelete.Header().Get("Set-Cookie"); !contains(setCookie, "session_id_local=;") {
		t.Errorf("Expected the session cookie to be cleared, got %q", setCookie)
	}

	// Step 3: the account and its data are gone
	if _, err := testServer.DB.GetUserByID(user.ID); !errors.Is(err, database.ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound after deletion, got %v", err)
	}
	if feeds, err := testServer.DB.GetUserFeeds(user.ID); err != nil || len(feeds) != 0 {
		t.Errorf("Expected no feeds after deletion, got %d (err %v)", len(feeds), err)
	}
	if records, _, err := testServer.DB.GetUserArticleHistoryPage(user.ID, "", 10); err != nil || len(records) != 0 {
		t.Errorf("Expected no article history after deletion, got %d (err %v)", len(records), err)
	}

	// Step 4: the session used for the deletion no longer authenticates
	reqAfter := helpers.CreateUnauthenticatedRequest(t, "GET", "/api/feeds", nil)
	for _, cookie := range reqDelete.Cookies() {
		reqAfter.AddCookie(cookie)
	}
	if rr := testServer.ExecuteRequest(reqAfter); rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 with the deleted account's session, got %d", rr.Code)
	}

	// Step 5: the other user's data is untouched
	if feeds, err := testServer.DB.GetUserFeeds(other.ID); err != nil || len(feeds) != 1 {
		t.Errorf("Expected other user's feed to remain, got %d (err %v)", len(feeds), err)
	}
	if records, _, err := testServer.DB.GetUserArticleHistoryPage(other.ID, "", 10); err != nil || len(records) != 1 {
		t.Errorf("Expected other user's article history to remain, got %d (err %v)", len(records), err)
	}

	// Step 6: a tombstone marks the Google account, so re-signing up does not start a new trial
	tombstoneID, err := secrets.AccountTombstoneID(context.Background(), user.GoogleID)
	if err != nil || tombstoneID == "" {
		t.Fatalf("Failed to compute tombstone ID: %q, %v", tombstoneID, err)
	}
	if found, err := testServer.DB.HasAccountTombstone(tombstoneID); err != nil || !found {
		t.Errorf("Expected a tombstone for the deleted account, got %v (err %v)", found, err)
	}
	otherTombstoneID, _ := secrets.AccountTombstoneID(context.Background(), other.GoogleID)
	if found, _ := testServer.DB.HasAccountTombstone(otherTombstoneID); found {
		t.Error("The other user should not have a tombstone")
	}
}
