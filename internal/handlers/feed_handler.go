package handlers

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"goread2/internal/auth"
	"goread2/internal/database"
	"goread2/internal/middleware"
	"goread2/internal/services"
)

type FeedHandler struct {
	feedService         *services.FeedService
	subscriptionService *services.SubscriptionService
	feedScheduler       *services.FeedScheduler
	db                  database.Database
}

func NewFeedHandler(feedService *services.FeedService, subscriptionService *services.SubscriptionService, feedScheduler *services.FeedScheduler, db database.Database) *FeedHandler {
	return &FeedHandler{
		feedService:         feedService,
		subscriptionService: subscriptionService,
		feedScheduler:       feedScheduler,
		db:                  db,
	}
}

func (fh *FeedHandler) GetFeeds(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	feeds, err := middleware.GetCachedUserFeeds(c, user.ID, fh.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Check if user can add more feeds
	if err := fh.subscriptionService.CanUserAddFeed(user.ID); err != nil {
		if errors.Is(err, services.ErrFeedLimitReached) {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":         "You've reached the limit of 20 feeds for free users. Upgrade to Pro for unlimited feeds.",
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	feed, err := fh.feedService.AddFeedForUser(user.ID, req.URL)
	if err != nil {
		log.Printf("Failed to add feed '%s' for user %d: %v", req.URL, user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, feed)
}

func (fh *FeedHandler) DeleteFeed(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feed ID"})
		return
	}

	if err := fh.feedService.UnsubscribeUserFromFeed(user.ID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Feed removed from your subscriptions successfully"})
}

func (fh *FeedHandler) GetArticles(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	idStr := c.Param("id")
	if idStr == "all" {
		// Parse pagination parameters
		limit := 50 // Default limit
		offset := 0 // Default offset
		unreadOnly := false // Default to showing all articles

		if limitStr := c.Query("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 200 {
				limit = parsedLimit
			}
		}

		if offsetStr := c.Query("offset"); offsetStr != "" {
			if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		if unreadStr := c.Query("unread_only"); unreadStr == "true" || unreadStr == "1" {
			unreadOnly = true
		}

		articles, err := fh.feedService.GetUserArticlesPaginated(user.ID, limit, offset, unreadOnly)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, articles)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feed ID"})
		return
	}

	articles, err := fh.feedService.GetUserFeedArticles(user.ID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, articles)
}

func (fh *FeedHandler) MarkRead(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	var req struct {
		IsRead bool `json:"is_read"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := fh.feedService.MarkUserArticleRead(user.ID, id, req.IsRead); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article updated successfully"})
}

func (fh *FeedHandler) ToggleStar(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	if err := fh.feedService.ToggleUserArticleStar(user.ID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article starred status toggled"})
}

func (fh *FeedHandler) RefreshFeeds(c *gin.Context) {
	// If this is the cron endpoint, verify it's authorized
	if c.Request.URL.Path == "/cron/refresh-feeds" {
		// In App Engine, verify the X-Appengine-Cron header
		if os.Getenv("GAE_ENV") == "standard" {
			cronHeader := c.GetHeader("X-Appengine-Cron")
			if cronHeader != "true" {
				log.Printf("Unauthorized cron request from IP: %s", auth.GetSecureClientIP(c))
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				return
			}
		} else {
			// In non-App Engine environments, require authentication with admin privileges
			user, exists := auth.GetUserFromContext(c)
			if !exists || !user.IsAdmin {
				log.Printf("Unauthorized cron request - requires admin authentication")
				c.JSON(http.StatusForbidden, gin.H{"error": "Admin authentication required"})
				return
			}
		}
		log.Printf("Cron feed refresh started at %v", time.Now())
	} else {
		log.Printf("Manual feed refresh started at %v", time.Now())
	}

	// Use staggered refresh if scheduler is available, otherwise fallback to regular refresh
	var err error
	if fh.feedScheduler != nil {
		err = fh.feedScheduler.RefreshFeedsStaggered()
	} else {
		err = fh.feedService.RefreshFeeds()
	}

	if err != nil {
		log.Printf("Feed refresh failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Feed refresh completed successfully at %v", time.Now())
	c.JSON(http.StatusOK, gin.H{"message": "Feeds refreshed successfully"})
}

func (fh *FeedHandler) DebugFeed(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feed ID"})
		return
	}

	// Get user feeds to verify subscription
	userFeeds, err := middleware.GetCachedUserFeeds(c, user.ID, fh.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user feeds", "details": err.Error()})
		return
	}

	// Check all articles for this feed (bypass user filtering for debug)
	allArticles, err := fh.feedService.GetArticles(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get all articles", "details": err.Error()})
		return
	}

	// Get user-specific articles
	userArticles, err := fh.feedService.GetUserFeedArticles(user.ID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user articles", "details": err.Error()})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL parameter required"})
		return
	}

	// Search for article by URL across all feeds
	article, err := fh.feedService.FindArticleByURL(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
		return
	}

	if article == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"found": false,
			"url":   url,
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get all feeds that appear in the UI
	userFeeds, err := middleware.GetCachedUserFeeds(c, user.ID, fh.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user feeds", "details": err.Error()})
		return
	}

	type FeedStatus struct {
		Feed            database.Feed `json:"feed"`
		IsSubscribed    bool          `json:"is_subscribed"`
		AllArticleCount int           `json:"all_article_count"`
		UserArticleCount int          `json:"user_article_count"`
		Status          string        `json:"status"`
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
			Feed:            feed,
			IsSubscribed:    isSubscribed,
			AllArticleCount: len(allArticles),
			UserArticleCount: len(userArticles),
			Status:          status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total_feeds": len(userFeeds),
		"feed_statuses": feedStatuses,
	})
}

func (fh *FeedHandler) GetUnreadCounts(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get cached user feeds first to avoid duplicate DB call in GetUserUnreadCounts
	userFeeds, err := middleware.GetCachedUserFeeds(c, user.ID, fh.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	unreadCounts, err := fh.feedService.GetUserUnreadCounts(user.ID, userFeeds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Cache headers are set by middleware for optimal performance
	c.JSON(http.StatusOK, unreadCounts)
}

func (fh *FeedHandler) ImportOPML(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Parse multipart form
	file, header, err := c.Request.FormFile("opml")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No OPML file provided"})
		return
	}
	defer func() { _ = file.Close() }()

	// Check file size (limit to 10MB)
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 10MB)"})
		return
	}

	// Read file content
	opmlData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read OPML file"})
		return
	}

	// Import feeds (this will check limits internally)
	importedCount, err := fh.feedService.ImportOPMLWithLimits(user.ID, opmlData, fh.subscriptionService)
	if err != nil {
		if errors.Is(err, services.ErrFeedLimitReached) {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":          "Import would exceed your feed limit of 20 feeds. Upgrade to Pro for unlimited feeds.",
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Generate OPML from user's feeds
	opmlData, err := fh.feedService.ExportOPML(user.ID)
	if err != nil {
		log.Printf("Failed to export OPML for user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OPML export"})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	subscriptionInfo, err := fh.subscriptionService.GetUserSubscriptionInfo(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, subscriptionInfo)
}

func (fh *FeedHandler) GetAccountStats(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get user feeds
	feeds, err := middleware.GetCachedUserFeeds(c, user.ID, fh.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get total articles count across all feeds
	totalArticles := 0
	totalUnread := 0
	activeFeeds := 0

	for _, feed := range feeds {
		articles, err := fh.feedService.GetUserFeedArticles(user.ID, feed.ID)
		if err != nil {
			continue // Skip failed feeds in stats
		}
		totalArticles += len(articles)

		unreadCount := 0
		for _, article := range articles {
			if !article.IsRead {
				unreadCount++
			}
		}
		totalUnread += unreadCount
		if unreadCount > 0 {
			activeFeeds++
		}
	}

	// Get subscription info for additional stats
	subscriptionInfo, _ := fh.subscriptionService.GetUserSubscriptionInfo(user.ID)

	stats := gin.H{
		"total_feeds":       len(feeds),
		"total_articles":    totalArticles,
		"total_unread":      totalUnread,
		"active_feeds":      activeFeeds,
		"subscription_info": subscriptionInfo,
		"feeds":             feeds,
	}

	c.JSON(http.StatusOK, stats)
}

func (fh *FeedHandler) UpdateMaxArticlesOnFeedAdd(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req struct {
		MaxArticles int `json:"max_articles" binding:"required,min=0,max=10000"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := fh.feedService.UpdateUserMaxArticlesOnFeedAdd(user.ID, req.MaxArticles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Setting updated successfully",
		"max_articles": req.MaxArticles,
	})
}
