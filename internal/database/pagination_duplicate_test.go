package database

import (
	"fmt"
	"testing"
	"time"
)

// TestCursorPaginationWithReadStateChanges tests the exact scenario described in gr-2:
// marking articles as read between pagination requests should NOT cause duplicates or skipped articles
func TestCursorPaginationWithReadStateChanges(t *testing.T) {
	db := setupTestDB(t)

	user := createTestUser(t, db)
	feed := createTestFeed(t, db)
	if err := db.SubscribeUserToFeed(user.ID, feed.ID); err != nil {
		t.Fatalf("SubscribeUserToFeed failed: %v", err)
	}

	// Create 100 articles to test pagination across multiple pages
	articles := make([]*Article, 100)
	baseTime := time.Now()
	for i := 0; i < 100; i++ {
		// Create articles with distinct timestamps (1 second apart) to avoid ties
		publishedAt := baseTime.Add(-time.Duration(i) * time.Second)
		article := &Article{
			FeedID:      feed.ID,
			Title:       "Article " + string(rune('A'+i/26)) + string(rune('A'+i%26)),
			URL:         fmt.Sprintf("https://example.com/article_%d", i),
			Content:     "Test content",
			Description: "Test description",
			PublishedAt: publishedAt,
			CreatedAt:   time.Now(),
		}
		if err := db.AddArticle(article); err != nil {
			t.Fatalf("AddArticle failed: %v", err)
		}
		articles[i] = article
	}

	// Step 1: Load first page of 50 articles with unread_only filter
	page1, err := db.GetUserArticlesPaginated(user.ID, 50, "", true)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated (page 1) failed: %v", err)
	}

	if len(page1.Articles) != 50 {
		t.Errorf("Expected 50 articles on page 1, got %d", len(page1.Articles))
	}

	if page1.NextCursor == "" {
		t.Fatal("Expected NextCursor to be set for page 1")
	}

	// Track IDs from page 1
	page1IDs := make(map[int]bool)
	for _, article := range page1.Articles {
		page1IDs[article.ID] = true
	}

	// Step 2: CRITICAL TEST - Mark some articles from page 1 as read
	// This simulates the user reading articles between pagination requests
	articlesToMarkRead := []int{
		page1.Articles[5].ID,   // Middle of page 1
		page1.Articles[15].ID,  // Middle of page 1
		page1.Articles[30].ID,  // Middle of page 1
		page1.Articles[45].ID,  // Near end of page 1
	}

	for _, articleID := range articlesToMarkRead {
		if err := db.MarkUserArticleRead(user.ID, articleID, true); err != nil {
			t.Fatalf("MarkUserArticleRead failed for article %d: %v", articleID, err)
		}
	}

	// Step 3: Load page 2 using the cursor from page 1
	page2, err := db.GetUserArticlesPaginated(user.ID, 50, page1.NextCursor, true)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated (page 2) failed: %v", err)
	}

	// Step 4: VERIFICATION - Check for duplicates
	duplicates := 0
	for _, article := range page2.Articles {
		if page1IDs[article.ID] {
			t.Errorf("DUPLICATE FOUND: Article ID %d appeared in both page 1 and page 2", article.ID)
			duplicates++
		}
	}

	if duplicates > 0 {
		t.Errorf("Found %d duplicate articles across pages", duplicates)
	}

	// Step 5: VERIFICATION - Check for skipped articles
	// Get ALL unread articles in one query (no pagination) to see what should have been returned
	allUnread, err := db.GetUserArticlesPaginated(user.ID, 200, "", true)
	if err != nil {
		t.Fatalf("GetUserArticlesPaginated (all) failed: %v", err)
	}

	// We marked 4 articles as read, so should have 96 unread
	expectedUnreadCount := 96
	if len(allUnread.Articles) != expectedUnreadCount {
		t.Logf("WARNING: Expected %d unread articles, got %d", expectedUnreadCount, len(allUnread.Articles))
	}

	// The first 50 unread articles should be from page 1
	// The next articles should match page 2
	allUnreadIDs := make(map[int]int) // ID -> position in full unread list
	for i, article := range allUnread.Articles {
		allUnreadIDs[article.ID] = i
	}

	// Check page 1 articles are in positions 0-49 (minus the 4 we marked read)
	page1PositionsValid := true
	for _, article := range page1.Articles {
		if !wasMarkedRead(article.ID, articlesToMarkRead) {
			pos, exists := allUnreadIDs[article.ID]
			if !exists {
				t.Errorf("Article %d from page 1 not found in all-unread query", article.ID)
			} else if pos >= 50 {
				t.Errorf("Article %d from page 1 is at position %d in full unread list (should be < 50)", article.ID, pos)
				page1PositionsValid = false
			}
		}
	}

	// Check page 2 articles start from around position 46 (50 - 4 marked read)
	// and continue sequentially
	if len(page2.Articles) > 0 {
		firstPage2Pos := allUnreadIDs[page2.Articles[0].ID]

		// After marking 4 articles as read from page 1, page 2 should start around position 46
		// (50 articles on page 1 - 4 marked as read = 46 unread articles before page 2 starts)
		expectedStartPos := 46
		if firstPage2Pos < expectedStartPos-2 || firstPage2Pos > expectedStartPos+2 {
			t.Errorf("Page 2 starts at position %d, expected around %d (±2). This suggests articles were SKIPPED",
				firstPage2Pos, expectedStartPos)
		}

		// Verify page 2 articles are sequential
		for i := 1; i < len(page2.Articles); i++ {
			prevPos := allUnreadIDs[page2.Articles[i-1].ID]
			currentPos := allUnreadIDs[page2.Articles[i].ID]

			if currentPos != prevPos+1 {
				t.Errorf("Page 2 articles not sequential: article at index %d is at position %d, previous was %d",
					i, currentPos, prevPos)
			}
		}
	}

	// Summary
	t.Logf("✓ Pagination test completed:")
	t.Logf("  - Page 1: %d articles", len(page1.Articles))
	t.Logf("  - Marked %d articles as read", len(articlesToMarkRead))
	t.Logf("  - Page 2: %d articles", len(page2.Articles))
	t.Logf("  - Total unread: %d", len(allUnread.Articles))
	t.Logf("  - Duplicates: %d", duplicates)

	if duplicates == 0 && page1PositionsValid {
		t.Log("✓ Cursor pagination working correctly - no duplicates or skips")
	}
}

// Helper function to check if an article ID was marked as read
func wasMarkedRead(articleID int, markedReadIDs []int) bool {
	for _, id := range markedReadIDs {
		if id == articleID {
			return true
		}
	}
	return false
}
