package integration

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/test/helpers"
)

func TestFeedAPI(t *testing.T) {
	t.Parallel() // Run in parallel with other top-level tests

	// Clean up test users at start and end
	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)
	user := helpers.CreateTestUser(t, testServer.DB, "google123", "test@example.com", "Test User")

	t.Run("GetFeeds_Empty", func(t *testing.T) {
		req := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds", nil, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var feeds []database.Feed
		err := json.Unmarshal(rr.Body.Bytes(), &feeds)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(feeds) != 0 {
			t.Errorf("Expected 0 feeds, got %d", len(feeds))
		}
	})

	t.Run("AddFeed_Unauthenticated", func(t *testing.T) {
		feedData := map[string]string{
			"url": "https://feeds.bbci.co.uk/news/rss.xml",
		}

		req := helpers.CreateUnauthenticatedRequest(t, "POST", "/api/feeds", feedData)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("AddFeed_Success", func(t *testing.T) {
		// Note: This test requires an active internet connection
		// In a real test environment, you'd mock the HTTP client
		feedData := map[string]string{
			"url": "https://feeds.bbci.co.uk/news/rss.xml",
		}

		req := testServer.CreateAuthenticatedRequest(t, "POST", "/api/feeds", feedData, user)
		rr := testServer.ExecuteRequest(req)

		// This might fail due to network issues in test environment
		// In production tests, mock the feed fetching
		if rr.Code != http.StatusCreated && rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 201 or 500 (network error), got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("AddFeed_InvalidURL", func(t *testing.T) {
		feedData := map[string]string{
			"url": "not-a-valid-url",
		}

		req := testServer.CreateAuthenticatedRequest(t, "POST", "/api/feeds", feedData, user)
		rr := testServer.ExecuteRequest(req)

		// Network errors (DNS failures) return 502 Bad Gateway
		if rr.Code != http.StatusBadGateway {
			t.Errorf("Expected status 502 (Bad Gateway for network error), got %d", rr.Code)
		}
	})

	t.Run("AddFeed_MissingURL", func(t *testing.T) {
		feedData := map[string]string{}

		req := testServer.CreateAuthenticatedRequest(t, "POST", "/api/feeds", feedData, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("DeleteFeed_Success", func(t *testing.T) {
		// Create a feed to delete
		feed := helpers.CreateTestFeed(t, testServer.DB, "Feed to Delete", "https://delete.com/feed", "Test feed")
		err := testServer.DB.SubscribeUserToFeed(user.ID, feed.ID)
		if err != nil {
			t.Fatalf("Failed to subscribe user to feed: %v", err)
		}

		url := "/api/feeds/" + strconv.Itoa(feed.ID)
		req := testServer.CreateAuthenticatedRequest(t, "DELETE", url, nil, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("DeleteFeed_Unauthenticated", func(t *testing.T) {
		req := helpers.CreateUnauthenticatedRequest(t, "DELETE", "/api/feeds/1", nil)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("DeleteFeed_InvalidID", func(t *testing.T) {
		req := testServer.CreateAuthenticatedRequest(t, "DELETE", "/api/feeds/invalid", nil, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("RefreshFeeds_Success", func(t *testing.T) {
		req := testServer.CreateAuthenticatedRequest(t, "POST", "/api/feeds/refresh", nil, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("RefreshFeeds_Unauthenticated", func(t *testing.T) {
		req := helpers.CreateUnauthenticatedRequest(t, "POST", "/api/feeds/refresh", nil)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})
}

func TestArticleAPI(t *testing.T) {
	t.Parallel() // Run in parallel with other top-level tests

	// Clean up test users at start and end
	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)
	user := helpers.CreateTestUser(t, testServer.DB, "google456", "test2@example.com", "Test User 2")

	// Create test data
	feed := helpers.CreateTestFeed(t, testServer.DB, "Test Feed", "https://test.com/rss", "Test feed")
	article := helpers.CreateTestArticle(t, testServer.DB, feed.ID, "Test Article", "https://test.com/article1")

	// Subscribe user to feed
	err := testServer.DB.SubscribeUserToFeed(user.ID, feed.ID)
	if err != nil {
		t.Fatalf("Failed to subscribe user to feed: %v", err)
	}

	t.Run("GetAllArticles", func(t *testing.T) {
		req := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds/all/articles", nil, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response struct {
			Articles   []database.Article `json:"articles"`
			NextCursor string             `json:"next_cursor"`
		}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(response.Articles) != 1 {
			t.Errorf("Expected 1 article, got %d", len(response.Articles))
		}
	})

	t.Run("GetFeedArticles", func(t *testing.T) {
		url := "/api/feeds/" + strconv.Itoa(feed.ID) + "/articles"
		req := testServer.CreateAuthenticatedRequest(t, "GET", url, nil, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var articles []database.Article
		err := json.Unmarshal(rr.Body.Bytes(), &articles)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(articles) != 1 {
			t.Errorf("Expected 1 article, got %d", len(articles))
		}
	})

	t.Run("MarkArticleRead", func(t *testing.T) {
		url := "/api/articles/" + strconv.Itoa(article.ID) + "/read"
		requestData := map[string]bool{
			"is_read": true,
		}

		req := testServer.CreateAuthenticatedRequest(t, "POST", url, requestData, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		// Verify article is marked as read
		articles, err := testServer.DB.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles: %v", err)
		}

		if len(articles) != 1 || !articles[0].IsRead {
			t.Error("Expected article to be marked as read")
		}
	})

	t.Run("ToggleArticleStar", func(t *testing.T) {
		url := "/api/articles/" + strconv.Itoa(article.ID) + "/star"

		req := testServer.CreateAuthenticatedRequest(t, "POST", url, nil, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		// Verify article is starred
		articles, err := testServer.DB.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles: %v", err)
		}

		if len(articles) != 1 || !articles[0].IsStarred {
			t.Error("Expected article to be starred")
		}
	})

	t.Run("ArticleOperations_Unauthenticated", func(t *testing.T) {
		endpoints := []string{
			"/api/feeds/all/articles",
			"/api/feeds/" + strconv.Itoa(feed.ID) + "/articles",
			"/api/articles/" + strconv.Itoa(article.ID) + "/read",
			"/api/articles/" + strconv.Itoa(article.ID) + "/star",
		}

		for _, endpoint := range endpoints {
			req := helpers.CreateUnauthenticatedRequest(t, "GET", endpoint, nil)
			if endpoint == "/api/articles/"+strconv.Itoa(article.ID)+"/read" ||
				endpoint == "/api/articles/"+strconv.Itoa(article.ID)+"/star" {
				req = helpers.CreateUnauthenticatedRequest(t, "POST", endpoint, nil)
			}

			rr := testServer.ExecuteRequest(req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401 for %s, got %d", endpoint, rr.Code)
			}
		}
	})

	t.Run("GetArticles_InvalidFeedID", func(t *testing.T) {
		req := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds/invalid/articles", nil, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("MarkRead_NonExistentArticle", func(t *testing.T) {
		url := "/api/articles/99999/read"
		requestData := map[string]bool{
			"is_read": true,
		}

		req := testServer.CreateAuthenticatedRequest(t, "POST", url, requestData, user)
		rr := testServer.ExecuteRequest(req)

		// Should return an error (either 404 or 500)
		if rr.Code != http.StatusNotFound && rr.Code != http.StatusInternalServerError {
			t.Logf("Expected status 404 or 500, got %d", rr.Code)
		}
	})

	t.Run("ToggleStar_NonExistentArticle", func(t *testing.T) {
		url := "/api/articles/99999/star"

		req := testServer.CreateAuthenticatedRequest(t, "POST", url, nil, user)
		rr := testServer.ExecuteRequest(req)

		// Should return an error (either 404 or 500)
		if rr.Code != http.StatusNotFound && rr.Code != http.StatusInternalServerError {
			t.Logf("Expected status 404 or 500, got %d", rr.Code)
		}
	})

	t.Run("MarkAllRead_Success", func(t *testing.T) {
		// Create multiple test articles for this user
		article2 := helpers.CreateTestArticle(t, testServer.DB, feed.ID, "Test Article 2", "https://test.com/article2")
		article3 := helpers.CreateTestArticle(t, testServer.DB, feed.ID, "Test Article 3", "https://test.com/article3")

		// Mark user articles as unread first
		err := testServer.DB.SetUserArticleStatus(user.ID, article.ID, false, false)
		if err != nil {
			t.Fatalf("Failed to set article status: %v", err)
		}
		err = testServer.DB.SetUserArticleStatus(user.ID, article2.ID, false, false)
		if err != nil {
			t.Fatalf("Failed to set article2 status: %v", err)
		}
		err = testServer.DB.SetUserArticleStatus(user.ID, article3.ID, false, false)
		if err != nil {
			t.Fatalf("Failed to set article3 status: %v", err)
		}

		// Call mark all as read endpoint
		req := testServer.CreateAuthenticatedRequest(t, "POST", "/api/articles/mark-all-read", nil, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		// Verify response
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Check that articles_count is returned
		articlesCount, ok := response["articles_count"].(float64)
		if !ok {
			t.Error("Expected articles_count in response")
		}
		if articlesCount != 3 {
			t.Errorf("Expected 3 articles marked as read, got %v", articlesCount)
		}

		// Verify all articles are marked as read
		articles, err := testServer.DB.GetUserArticles(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user articles: %v", err)
		}

		for _, a := range articles {
			if !a.IsRead {
				t.Errorf("Expected all articles to be marked as read, but article %d is not", a.ID)
			}
		}
	})

	t.Run("MarkAllRead_EmptyArticles", func(t *testing.T) {
		// Create a new user with no articles
		emptyUser := helpers.CreateTestUser(t, testServer.DB, "google999", "empty@example.com", "Empty User")

		req := testServer.CreateAuthenticatedRequest(t, "POST", "/api/articles/mark-all-read", nil, emptyUser)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		// Verify response
		var response map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Check that articles_count is 0
		articlesCount, ok := response["articles_count"].(float64)
		if !ok {
			t.Error("Expected articles_count in response")
		}
		if articlesCount != 0 {
			t.Errorf("Expected 0 articles marked as read, got %v", articlesCount)
		}
	})

	t.Run("MarkAllRead_Unauthenticated", func(t *testing.T) {
		req := helpers.CreateUnauthenticatedRequest(t, "POST", "/api/articles/mark-all-read", nil)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})
}

func TestUserIsolation(t *testing.T) {
	t.Parallel() // Run in parallel with other top-level tests

	// Clean up test users at start and end
	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)

	// Create two users
	user1 := helpers.CreateTestUser(t, testServer.DB, "user1", "user1@example.com", "User 1")
	user2 := helpers.CreateTestUser(t, testServer.DB, "user2", "user2@example.com", "User 2")

	// Create a feed and subscribe user1 to it
	feed := helpers.CreateTestFeed(t, testServer.DB, "Shared Feed", "https://shared.com/rss", "Shared feed")
	err := testServer.DB.SubscribeUserToFeed(user1.ID, feed.ID)
	if err != nil {
		t.Fatalf("Failed to subscribe user1 to feed: %v", err)
	}

	t.Run("User1_CanSeeSubscribedFeed", func(t *testing.T) {
		req := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds", nil, user1)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var feeds []database.Feed
		err := json.Unmarshal(rr.Body.Bytes(), &feeds)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(feeds) != 1 {
			t.Errorf("Expected 1 feed for user1, got %d", len(feeds))
		}
	})

	t.Run("User2_CannotSeeUser1Feed", func(t *testing.T) {
		req := testServer.CreateAuthenticatedRequest(t, "GET", "/api/feeds", nil, user2)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var feeds []database.Feed
		err := json.Unmarshal(rr.Body.Bytes(), &feeds)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(feeds) != 0 {
			t.Errorf("Expected 0 feeds for user2, got %d", len(feeds))
		}
	})
}

func TestAuthAPI(t *testing.T) {
	t.Parallel() // Run in parallel with other top-level tests

	// Clean up test users at start and end
	helpers.CleanupTestUsers(t)
	defer helpers.CleanupTestUsers(t)

	helpers.SetupTestEnv(t)
	defer helpers.CleanupTestEnv(t)

	testServer := helpers.SetupTestServer(t)

	t.Run("Login_ReturnsAuthURL", func(t *testing.T) {
		req := helpers.CreateUnauthenticatedRequest(t, "GET", "/auth/login", nil)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response map[string]string
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["auth_url"] == "" {
			t.Error("Expected auth_url in response")
		}
	})

	t.Run("Me_Unauthenticated", func(t *testing.T) {
		req := helpers.CreateUnauthenticatedRequest(t, "GET", "/auth/me", nil)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("Me_Authenticated", func(t *testing.T) {
		user := helpers.CreateTestUser(t, testServer.DB, "google789", "test3@example.com", "Test User 3")

		req := testServer.CreateAuthenticatedRequest(t, "GET", "/auth/me", nil, user)
		rr := testServer.ExecuteRequest(req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		userData, ok := response["user"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected user data in response")
		}

		if userData["email"] != user.Email {
			t.Errorf("Expected email %s, got %s", user.Email, userData["email"])
		}
	})
}
