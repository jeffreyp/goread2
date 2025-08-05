package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"goread2/internal/auth"
	"goread2/internal/database"
	"goread2/internal/handlers"
	"goread2/internal/services"
)

func main() {
	log.Printf("=== GOREAD2 STARTING UP ===")
	
	// Debug OAuth environment variables
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	
	log.Printf("Environment debug:")
	if googleClientID == "" {
		log.Printf("  GOOGLE_CLIENT_ID: (EMPTY/NOT SET)")
	} else {
		log.Printf("  GOOGLE_CLIENT_ID: %s", googleClientID)
	}
	
	if googleClientSecret == "" {
		log.Printf("  GOOGLE_CLIENT_SECRET: (EMPTY/NOT SET)")
	} else {
		log.Printf("  GOOGLE_CLIENT_SECRET: %s", googleClientSecret)
	}
	
	log.Printf("  GAE_ENV: %s", os.Getenv("GAE_ENV"))
	log.Printf("  PORT: %s", os.Getenv("PORT"))
	log.Printf("  GOOGLE_CLOUD_PROJECT: %s", os.Getenv("GOOGLE_CLOUD_PROJECT"))
	
	log.Printf("Initializing database...")
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	log.Printf("Database initialized successfully")

	// Initialize services
	log.Printf("Initializing services...")
	feedService := services.NewFeedService(db)
	authService := auth.NewAuthService(db)
	sessionManager := auth.NewSessionManager(db)
	log.Printf("Services initialized")

	// Validate OAuth configuration
	log.Printf("Validating OAuth configuration...")
	if err := authService.ValidateConfig(); err != nil {
		log.Fatal("OAuth configuration error:", err)
	}
	log.Printf("OAuth configuration valid")

	// Initialize handlers
	log.Printf("Initializing handlers...")
	feedHandler := handlers.NewFeedHandler(feedService)
	authHandler := handlers.NewAuthHandler(authService, sessionManager)
	log.Printf("Handlers initialized")

	// Initialize middleware
	log.Printf("Initializing middleware...")
	authMiddleware := auth.NewMiddleware(sessionManager)
	log.Printf("Middleware initialized")

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

	// Public routes
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "GoRead2 - RSS Reader",
		})
	})

	r.GET("/privacy", func(c *gin.Context) {
		c.HTML(http.StatusOK, "privacy.html", gin.H{
			"title": "Privacy Policy - GoRead2",
		})
	})

	// Auth routes (public)
	auth := r.Group("/auth")
	{
		auth.GET("/login", authHandler.Login)
		auth.GET("/callback", authHandler.Callback)
		auth.POST("/logout", authHandler.Logout)
		auth.GET("/me", authMiddleware.OptionalAuth(), authHandler.Me)
	}

	// Protected API routes
	api := r.Group("/api")
	api.Use(authMiddleware.RequireAuth())
	{
		api.GET("/feeds", feedHandler.GetFeeds)
		api.POST("/feeds", feedHandler.AddFeed)
		api.DELETE("/feeds/:id", feedHandler.DeleteFeed)
		api.GET("/feeds/:id/articles", feedHandler.GetArticles)
		api.GET("/feeds/:id/debug", feedHandler.DebugFeed)
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
