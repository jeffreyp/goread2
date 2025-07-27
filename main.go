package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"goread2/internal/database"
	"goread2/internal/handlers"
	"goread2/internal/services"
)

func main() {
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	feedService := services.NewFeedService(db)
	feedHandler := handlers.NewFeedHandler(feedService)

	// Set up Gin with appropriate settings for App Engine
	if os.Getenv("GAE_ENV") == "standard" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*")
	
	// Static files are handled by app.yaml in App Engine
	if os.Getenv("GAE_ENV") != "standard" {
		r.Static("/static", "./web/static")
	}

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "GoRead2 - RSS Reader",
		})
	})

	api := r.Group("/api")
	{
		api.GET("/feeds", feedHandler.GetFeeds)
		api.POST("/feeds", feedHandler.AddFeed)
		api.DELETE("/feeds/:id", feedHandler.DeleteFeed)
		api.GET("/feeds/:id/articles", feedHandler.GetArticles)
		api.POST("/articles/:id/read", feedHandler.MarkRead)
		api.POST("/articles/:id/star", feedHandler.ToggleStar)
		api.POST("/feeds/refresh", feedHandler.RefreshFeeds)
	}

	// Get port from environment (App Engine sets this)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(r.Run(":" + port))
}