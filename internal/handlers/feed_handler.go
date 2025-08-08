package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"goread2/internal/auth"
	"goread2/internal/database"
	"goread2/internal/services"
)

type FeedHandler struct {
	feedService *services.FeedService
}

func NewFeedHandler(feedService *services.FeedService) *FeedHandler {
	return &FeedHandler{feedService: feedService}
}

func (fh *FeedHandler) GetFeeds(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		log.Printf("GetFeeds: No user in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	log.Printf("GetFeeds: Getting feeds for user %d", user.ID)
	feeds, err := fh.feedService.GetUserFeeds(user.ID)
	if err != nil {
		log.Printf("GetFeeds: ERROR getting feeds for user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Debug logging
	log.Printf("GetFeeds: Found %d feeds for user %d", len(feeds), user.ID)
	for i, feed := range feeds {
		log.Printf("GetFeeds: Feed %d: ID=%d, Title=%s", i+1, feed.ID, feed.Title)
	}
	
	// Ensure we return an empty array instead of null
	if feeds == nil {
		feeds = []database.Feed{}
		log.Printf("GetFeeds: feeds was nil, converted to empty slice")
	}
	
	log.Printf("GetFeeds: Returning %d feeds as JSON", len(feeds))
	c.JSON(http.StatusOK, feeds)
}

func (fh *FeedHandler) AddFeed(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("AddFeed: User %d adding feed %s", user.ID, req.URL)
	
	feed, err := fh.feedService.AddFeedForUser(user.ID, req.URL)
	if err != nil {
		log.Printf("AddFeed: Error adding feed for user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("AddFeed: Successfully added feed %d (%s) for user %d", feed.ID, feed.Title, user.ID)
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

	log.Printf("DeleteFeed: Unsubscribing user %d from feed %d", user.ID, id)
	if err := fh.feedService.UnsubscribeUserFromFeed(user.ID, id); err != nil {
		log.Printf("DeleteFeed: Error unsubscribing user %d from feed %d: %v", user.ID, id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("DeleteFeed: Successfully unsubscribed user %d from feed %d", user.ID, id)
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
		articles, err := fh.feedService.GetUserArticles(user.ID)
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
	if err := fh.feedService.RefreshFeeds(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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
	userFeeds, err := fh.feedService.GetUserFeeds(user.ID)
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
		"user_id":              user.ID,
		"feed_id":              id,
		"is_subscribed":        isSubscribed,
		"user_feeds_count":     len(userFeeds),
		"all_articles_count":   len(allArticles),
		"user_articles_count":  len(userArticles),
		"user_feeds":           userFeeds,
		"all_articles":         allArticles,
		"user_articles":        userArticles,
	})
}

func (fh *FeedHandler) GetUnreadCounts(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	fmt.Printf("GetUnreadCounts: Called for user %d\n", user.ID)
	unreadCounts, err := fh.feedService.GetUserUnreadCounts(user.ID)
	if err != nil {
		fmt.Printf("GetUnreadCounts: Error for user %d: %v\n", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("GetUnreadCounts: Returning counts for user %d: %+v\n", user.ID, unreadCounts)
	c.JSON(http.StatusOK, unreadCounts)
}
