package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/services"
)

type ArticleHandler struct {
	feedService *services.FeedService
}

func NewArticleHandler(feedService *services.FeedService) *ArticleHandler {
	return &ArticleHandler{feedService: feedService}
}

func (ah *ArticleHandler) GetArticle(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The article ID is not valid."})
		return
	}

	article, err := ah.feedService.GetArticleByID(user.ID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve the article. Please try again."})
		return
	}
	if article == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "The requested article could not be found."})
		return
	}

	c.JSON(http.StatusOK, article)
}
