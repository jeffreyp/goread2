package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/database"
	"github.com/jeffreyp/goread2/internal/middleware"
	"github.com/jeffreyp/goread2/internal/services"
)

// TaskQueue enqueues async work for a task handler endpoint to process. It is
// satisfied by *services.CloudTasksQueue; tests and environments without
// Cloud Tasks configured leave it nil, which falls back to running cron work
// synchronously/in-process instead of failing.
type TaskQueue interface {
	Enqueue(ctx context.Context, relativeURI string) error
}

type FeedHandler struct {
	feedService         *services.FeedService
	subscriptionService *services.SubscriptionService
	feedScheduler       *services.FeedScheduler
	db                  database.Database
	taskQueue           TaskQueue
}

func NewFeedHandler(feedService *services.FeedService, subscriptionService *services.SubscriptionService, feedScheduler *services.FeedScheduler, db database.Database) *FeedHandler {
	return &FeedHandler{
		feedService:         feedService,
		subscriptionService: subscriptionService,
		feedScheduler:       feedScheduler,
		db:                  db,
	}
}

// SetTaskQueue wires up async dispatch for the cron endpoints. Called once
// from main after construction, only when Cloud Tasks is configured
// (production App Engine); left unset, cron handlers keep doing their own
// in-process work.
func (fh *FeedHandler) SetTaskQueue(tq TaskQueue) {
	fh.taskQueue = tq
}

func (fh *FeedHandler) GetFeeds(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	feeds, err := middleware.GetCachedUserFeeds(c, user.ID, fh.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve your feeds. Please try again."})
		return
	}

	// Ensure we return an empty array instead of null
	if feeds == nil {
		feeds = []database.Feed{}
	}

	// Cache headers are set by middleware for optimal performance
	c.JSON(http.StatusOK, feeds)
}

func (fh *FeedHandler) AddFeed(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	// Check if user can add more feeds
	if err := fh.subscriptionService.CanUserAddFeed(user.ID); err != nil {
		if errors.Is(err, services.ErrFeedLimitReached) {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":         fmt.Sprintf("You've reached the limit of %d feeds for free users. Upgrade to Pro for unlimited feeds.", services.FreeTrialFeedLimit),
				"limit_reached": true,
				"current_limit": services.FreeTrialFeedLimit,
			})
			return
		}
		if errors.Is(err, services.ErrTrialExpired) {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":         "Your 30-day free trial has expired. Subscribe to continue using GoRead2.",
				"trial_expired": true,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred. Please try again."})
		return
	}

	var req struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The request body could not be parsed."})
		return
	}

	feed, err := fh.feedService.AddFeedForUser(user.ID, req.URL)
	if err != nil {
		log.Printf("Failed to add feed '%s' for user %d: %v", req.URL, user.ID, err)

		// Get structured error details
		errorDetails := services.GetErrorDetails(err)

		// Map error types to HTTP status codes
		var statusCode int
		switch {
		case errors.Is(err, services.ErrInvalidURL):
			statusCode = http.StatusBadRequest
		case errors.Is(err, services.ErrSSRFBlocked):
			statusCode = http.StatusBadRequest
		case errors.Is(err, services.ErrFeedNotFound):
			statusCode = http.StatusNotFound
		case errors.Is(err, services.ErrFeedTimeout):
			statusCode = http.StatusRequestTimeout
		case errors.Is(err, services.ErrInvalidFeedFormat):
			statusCode = http.StatusUnprocessableEntity
		case errors.Is(err, services.ErrNetworkError):
			statusCode = http.StatusBadGateway
		case errors.Is(err, services.ErrDatabaseError):
			statusCode = http.StatusInternalServerError
		default:
			statusCode = http.StatusInternalServerError
		}

		c.JSON(statusCode, errorDetails)
		return
	}
	middleware.InvalidateCachedUserFeeds(c, user.ID)
	c.JSON(http.StatusCreated, feed)
}

func (fh *FeedHandler) DeleteFeed(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The feed ID is not valid."})
		return
	}

	if err := fh.feedService.UnsubscribeUserFromFeed(user.ID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove the feed. Please try again."})
		return
	}
	middleware.InvalidateCachedUserFeeds(c, user.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Feed removed from your subscriptions successfully"})
}

// parseArticlePaginationParams reads the limit/cursor/unread_only query parameters shared by
// the "all articles" and per-feed article listing endpoints.
func parseArticlePaginationParams(c *gin.Context) (limit int, cursor string, unreadOnly bool) {
	limit = 50 // Default limit

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	cursor = c.Query("cursor") // Get cursor from query parameter

	if unreadStr := c.Query("unread_only"); unreadStr == "true" || unreadStr == "1" {
		unreadOnly = true
	}

	return limit, cursor, unreadOnly
}

func (fh *FeedHandler) GetArticles(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	idStr := c.Param("id")
	limit, cursor, unreadOnly := parseArticlePaginationParams(c)

	if idStr == "all" {
		result, err := fh.feedService.GetUserArticlesPaginated(user.ID, limit, cursor, unreadOnly)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve your articles. Please try again."})
			return
		}

		// Return both articles and next_cursor for pagination
		c.JSON(http.StatusOK, gin.H{
			"articles":    result.Articles,
			"next_cursor": result.NextCursor,
		})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The feed ID is not valid."})
		return
	}

	result, err := fh.feedService.GetUserFeedArticlesPaginated(user.ID, id, limit, cursor, unreadOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve articles for this feed. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"articles":    result.Articles,
		"next_cursor": result.NextCursor,
	})
}

func (fh *FeedHandler) MarkRead(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The article ID is not valid."})
		return
	}

	var req struct {
		IsRead  bool `json:"is_read"`
		FeedID  int  `json:"feed_id"`
		WasRead bool `json:"was_read"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The request body could not be parsed."})
		return
	}

	if err := fh.feedService.MarkUserArticleRead(user.ID, id, req.IsRead, req.FeedID, req.WasRead); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update the article. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article updated successfully"})
}

func (fh *FeedHandler) ToggleStar(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The article ID is not valid."})
		return
	}

	if err := fh.feedService.ToggleUserArticleStar(user.ID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update the article. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article starred status toggled"})
}

func (fh *FeedHandler) MarkAllRead(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	count, err := fh.feedService.MarkAllArticlesRead(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark articles as read. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "All articles marked as read",
		"articles_count": count,
	})
}

func (fh *FeedHandler) RefreshFeeds(c *gin.Context) {
	// If this is the cron endpoint, verify it's authorized
	if c.Request.URL.Path == "/cron/refresh-feeds" {
		if !auth.VerifyCronRequest(c) {
			return
		}
		if fh.taskQueue != nil {
			// Enqueue and return immediately; Cloud Tasks dispatches to
			// /tasks/refresh-feeds and owns retries, so the work survives
			// this instance shutting down mid-refresh.
			if err := fh.taskQueue.Enqueue(c.Request.Context(), "/tasks/refresh-feeds"); err != nil {
				log.Printf("Failed to enqueue feed refresh task: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue feed refresh"})
				return
			}
			c.JSON(http.StatusAccepted, gin.H{"message": "Feed refresh enqueued"})
			return
		}
		// No task queue configured (local dev): fall back to a background
		// goroutine so GAE doesn't hold the instance. Not guaranteed to
		// complete if the instance is killed mid-refresh; cron retries are
		// the only safety net in this path.
		log.Printf("Cron feed refresh started at %v (background)", time.Now())
		go func() {
			if err := fh.doRefreshFeeds(); err != nil {
				log.Printf("Cron feed refresh failed: %v", err)
			} else {
				log.Printf("Cron feed refresh completed at %v", time.Now())
			}
		}()
		c.JSON(http.StatusAccepted, gin.H{"message": "Feed refresh started"})
		return
	}

	// Manual (API) path: run synchronously so the caller gets a result.
	log.Printf("Manual feed refresh started at %v", time.Now())
	if err := fh.doRefreshFeeds(); err != nil {
		log.Printf("Feed refresh failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh feeds. Please try again."})
		return
	}

	log.Printf("Feed refresh completed successfully at %v", time.Now())
	c.JSON(http.StatusOK, gin.H{"message": "Feeds refreshed successfully"})
}

// TaskRefreshFeeds is the Cloud Tasks worker endpoint for /tasks/refresh-feeds.
// It performs the same work RefreshFeeds does for the cron path, but runs
// synchronously so its response (success or failure) reports the outcome to
// Cloud Tasks, which retries on non-2xx per the queue's retry policy.
func (fh *FeedHandler) TaskRefreshFeeds(c *gin.Context) {
	if !auth.VerifyTaskRequest(c) {
		return
	}

	log.Printf("Task feed refresh started at %v", time.Now())
	if err := fh.doRefreshFeeds(); err != nil {
		log.Printf("Task feed refresh failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh feeds"})
		return
	}

	log.Printf("Task feed refresh completed at %v", time.Now())
	c.JSON(http.StatusOK, gin.H{"message": "Feeds refreshed successfully"})
}

func (fh *FeedHandler) doRefreshFeeds() error {
	if fh.feedScheduler != nil {
		return fh.feedScheduler.RefreshFeedsStaggered()
	}
	return fh.feedService.RefreshFeeds()
}

func (fh *FeedHandler) CleanupOrphanedUserArticles(c *gin.Context) {
	// If this is the cron endpoint, verify it's authorized
	if c.Request.URL.Path == "/cron/cleanup-orphaned-articles" {
		if !auth.VerifyCronRequest(c) {
			return
		}
		if fh.taskQueue != nil {
			if err := fh.taskQueue.Enqueue(c.Request.Context(), "/tasks/cleanup-orphaned-articles"); err != nil {
				log.Printf("Failed to enqueue cleanup task: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue cleanup"})
				return
			}
			c.JSON(http.StatusAccepted, gin.H{"message": "Cleanup enqueued"})
			return
		}
		log.Printf("Cron cleanup started at %v", time.Now())
	} else {
		log.Printf("Manual cleanup started at %v", time.Now())
	}

	deletedCount, err := fh.doCleanupOrphanedArticles()
	if err != nil {
		log.Printf("Cleanup failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean up orphaned articles. Please try again."})
		return
	}

	log.Printf("Cleanup completed successfully at %v, deleted %d orphaned records", time.Now(), deletedCount)
	c.JSON(http.StatusOK, gin.H{
		"message":       "Cleanup completed successfully",
		"deleted_count": deletedCount,
	})
}

// TaskCleanupOrphanedArticles is the Cloud Tasks worker endpoint for
// /tasks/cleanup-orphaned-articles. See TaskRefreshFeeds for why this runs
// synchronously rather than backgrounding the work.
func (fh *FeedHandler) TaskCleanupOrphanedArticles(c *gin.Context) {
	if !auth.VerifyTaskRequest(c) {
		return
	}

	log.Printf("Task cleanup started at %v", time.Now())
	deletedCount, err := fh.doCleanupOrphanedArticles()
	if err != nil {
		log.Printf("Task cleanup failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean up orphaned articles"})
		return
	}

	log.Printf("Task cleanup completed at %v, deleted %d orphaned records", time.Now(), deletedCount)
	c.JSON(http.StatusOK, gin.H{
		"message":       "Cleanup completed successfully",
		"deleted_count": deletedCount,
	})
}

func (fh *FeedHandler) doCleanupOrphanedArticles() (int, error) {
	// Orphaned for more than 7 days.
	return fh.db.CleanupOrphanedUserArticles(7)
}

func (fh *FeedHandler) DebugFeed(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The feed ID is not valid."})
		return
	}

	// Get user feeds to verify subscription
	userFeeds, err := middleware.GetCachedUserFeeds(c, user.ID, fh.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve your feeds. Please try again.", "details": err.Error()})
		return
	}

	// Check all articles for this feed (bypass user filtering for debug)
	allArticles, err := fh.feedService.GetArticles(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve articles for this feed. Please try again.", "details": err.Error()})
		return
	}

	// Get user-specific articles
	userArticles, err := fh.feedService.GetUserFeedArticles(user.ID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve articles for this feed. Please try again.", "details": err.Error()})
		return
	}

	// Check if user is subscribed to this feed
	isSubscribed := false
	for _, feed := range userFeeds {
		if feed.ID == id {
			isSubscribed = true
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":             user.ID,
		"feed_id":             id,
		"is_subscribed":       isSubscribed,
		"user_feeds_count":    len(userFeeds),
		"all_articles_count":  len(allArticles),
		"user_articles_count": len(userArticles),
		"user_feeds":          userFeeds,
		"all_articles":        allArticles,
		"user_articles":       userArticles,
	})
}

func (fh *FeedHandler) DebugArticleByURL(c *gin.Context) {
	_, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A URL parameter is required."})
		return
	}

	// Search for article by URL across all feeds
	article, err := fh.feedService.FindArticleByURL(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "A database error occurred.", "details": err.Error()})
		return
	}

	if article == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"found":   false,
			"url":     url,
			"message": "Article not found in database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"found":   true,
		"url":     url,
		"article": article,
	})
}

func (fh *FeedHandler) DebugAllSubscriptions(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	// Get all feeds that appear in the UI
	userFeeds, err := middleware.GetCachedUserFeeds(c, user.ID, fh.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve your feeds. Please try again.", "details": err.Error()})
		return
	}

	type FeedStatus struct {
		Feed             database.Feed `json:"feed"`
		IsSubscribed     bool          `json:"is_subscribed"`
		AllArticleCount  int           `json:"all_article_count"`
		UserArticleCount int           `json:"user_article_count"`
		Status           string        `json:"status"`
	}

	var feedStatuses []FeedStatus

	for _, feed := range userFeeds {
		// Check subscription status
		allArticles, err := fh.feedService.GetArticles(feed.ID)
		if err != nil {
			allArticles = []database.Article{}
		}

		userArticles, err := fh.feedService.GetUserFeedArticles(user.ID, feed.ID)
		if err != nil {
			userArticles = []database.Article{}
		}

		isSubscribed := len(userArticles) > 0 || len(allArticles) == 0
		status := "OK"
		if len(allArticles) > 0 && len(userArticles) == 0 {
			isSubscribed = false
			status = "BROKEN - Feed shows in UI but no access to articles"
		}

		feedStatuses = append(feedStatuses, FeedStatus{
			Feed:             feed,
			IsSubscribed:     isSubscribed,
			AllArticleCount:  len(allArticles),
			UserArticleCount: len(userArticles),
			Status:           status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total_feeds":   len(userFeeds),
		"feed_statuses": feedStatuses,
	})
}

func (fh *FeedHandler) GetUnreadCounts(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	// Get cached user feeds first to avoid duplicate DB call in GetUserUnreadCounts
	userFeeds, err := middleware.GetCachedUserFeeds(c, user.ID, fh.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve your feeds. Please try again."})
		return
	}

	unreadCounts, err := fh.feedService.GetUserUnreadCounts(user.ID, userFeeds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve unread counts. Please try again."})
		return
	}

	// Cache headers are set by middleware for optimal performance
	c.JSON(http.StatusOK, unreadCounts)
}

func (fh *FeedHandler) ImportOPML(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	// Parse multipart form
	file, header, err := c.Request.FormFile("opml")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No OPML file was included in the request."})
		return
	}
	defer func() { _ = file.Close() }()

	// Check file size (limit to 10MB)
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The file exceeds the maximum allowed size of 10 MB."})
		return
	}

	// Read file content
	opmlData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "The OPML file could not be read."})
		return
	}

	// Import feeds (this will check limits internally)
	importedCount, err := fh.feedService.ImportOPMLWithLimits(user.ID, opmlData, fh.subscriptionService)
	if err != nil {
		if errors.Is(err, services.ErrFeedLimitReached) {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":          fmt.Sprintf("Import would exceed your feed limit of %d feeds. Upgrade to Pro for unlimited feeds.", services.FreeTrialFeedLimit),
				"limit_reached":  true,
				"current_limit":  services.FreeTrialFeedLimit,
				"imported_count": importedCount,
			})
			return
		}
		if errors.Is(err, services.ErrTrialExpired) {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":         "Your 30-day free trial has expired. Subscribe to continue using GoRead2.",
				"trial_expired": true,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal error occurred while importing. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "OPML imported successfully",
		"imported_count": importedCount,
	})
}

func (fh *FeedHandler) ExportOPML(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	// Generate OPML from user's feeds
	opmlData, err := fh.feedService.ExportOPML(user.ID)
	if err != nil {
		log.Printf("Failed to export OPML for user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate the OPML export. Please try again."})
		return
	}

	// Set headers for file download
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=goread2-subscriptions.opml")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")

	// Return the OPML data
	c.Data(http.StatusOK, "application/xml; charset=utf-8", opmlData)
}

func (fh *FeedHandler) GetSubscriptionInfo(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	subscriptionInfo, err := fh.subscriptionService.GetUserSubscriptionInfo(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subscription information. Please try again."})
		return
	}

	c.JSON(http.StatusOK, subscriptionInfo)
}

func (fh *FeedHandler) GetAccountStats(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	// Get user feeds
	feeds, err := middleware.GetCachedUserFeeds(c, user.ID, fh.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve your feeds. Please try again."})
		return
	}

	accountStats, err := fh.feedService.GetAccountStats(user.ID, feeds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve account statistics. Please try again."})
		return
	}

	// Get subscription info for additional stats
	subscriptionInfo, _ := fh.subscriptionService.GetUserSubscriptionInfo(user.ID)

	// Combine all stats
	stats := gin.H{
		"total_feeds":       len(feeds),
		"total_articles":    accountStats["total_articles"],
		"total_unread":      accountStats["total_unread"],
		"active_feeds":      accountStats["active_feeds"],
		"subscription_info": subscriptionInfo,
		"feeds":             feeds,
	}

	c.JSON(http.StatusOK, stats)
}

func (fh *FeedHandler) UpdateMaxArticlesOnFeedAdd(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	var req struct {
		MaxArticles int `json:"max_articles" binding:"min=0,max=10000"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The request body could not be parsed."})
		return
	}

	if err := fh.feedService.UpdateUserMaxArticlesOnFeedAdd(user.ID, req.MaxArticles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update the setting. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Setting updated successfully",
		"max_articles": req.MaxArticles,
	})
}
