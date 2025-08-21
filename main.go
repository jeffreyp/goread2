package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"goread2/internal/auth"
	"goread2/internal/config"
	"goread2/internal/database"
	"goread2/internal/handlers"
	"goread2/internal/services"
)

func main() {
	// Load configuration
	cfg := config.Load()
	log.Printf("Subscription system enabled: %v", cfg.SubscriptionEnabled)

	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize services
	feedService := services.NewFeedService(db)
	subscriptionService := services.NewSubscriptionService(db)
	authService := auth.NewAuthService(db)
	sessionManager := auth.NewSessionManager(db)

	// Validate OAuth configuration
	if err := authService.ValidateConfig(); err != nil {
		log.Fatal("OAuth configuration error:", err)
	}

	// Initialize payment service and validate Stripe configuration only if subscriptions are enabled
	var paymentService *services.PaymentService
	if cfg.SubscriptionEnabled {
		paymentService = services.NewPaymentService(db, subscriptionService)
		
		// Validate Stripe configuration (optional - only if Stripe keys are provided)
		if cfg.StripeSecretKey != "" {
			if err := paymentService.ValidateStripeConfig(); err != nil {
				log.Printf("Warning: Stripe configuration incomplete: %v", err)
				log.Println("Subscription features will be disabled")
			}
		}
	} else {
		log.Println("Subscription system is disabled")
	}

	// Initialize handlers
	feedHandler := handlers.NewFeedHandler(feedService, subscriptionService)
	authHandler := handlers.NewAuthHandler(authService, sessionManager)
	var paymentHandler *handlers.PaymentHandler
	if cfg.SubscriptionEnabled && paymentService != nil {
		paymentHandler = handlers.NewPaymentHandler(paymentService)
	}

	// Initialize middleware
	authMiddleware := auth.NewMiddleware(sessionManager)

	// Set up Gin with appropriate settings for App Engine
	if os.Getenv("GAE_ENV") == "standard" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	
	// Add gzip compression for all responses
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	
	// Add caching headers middleware
	r.Use(func(c *gin.Context) {
		// Cache static assets for 24 hours with versioning
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "public, max-age=86400, immutable")
			c.Header("ETag", "\"static-v2\"")
			c.Header("Vary", "Accept-Encoding")
		}
		// Cache API responses for 60 seconds
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Cache-Control", "private, max-age=60")
		}
		// Cache HTML pages for 5 minutes
		if c.Request.URL.Path == "/" || strings.HasSuffix(c.Request.URL.Path, ".html") {
			c.Header("Cache-Control", "public, max-age=300")
		}
		c.Next()
	})
	
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

	// Protected pages
	r.GET("/account", authMiddleware.RequireAuth(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "account.html", gin.H{
			"title": "Account Management - GoRead2",
		})
	})

	// Subscription success/cancel pages (public) - only if subscriptions are enabled
	if cfg.SubscriptionEnabled && paymentHandler != nil {
		r.GET("/subscription/success", paymentHandler.SubscriptionSuccess)
		r.GET("/subscription/cancel", paymentHandler.SubscriptionCancel)
	}

	// Auth routes (public)
	auth := r.Group("/auth")
	{
		auth.GET("/login", authHandler.Login)
		auth.GET("/callback", authHandler.Callback)
		auth.POST("/logout", authHandler.Logout)
		auth.GET("/me", authMiddleware.OptionalAuth(), authHandler.Me)
	}

	// Public cron endpoint (no auth required)
	r.GET("/cron/refresh-feeds", feedHandler.RefreshFeeds)
	r.POST("/cron/refresh-feeds", feedHandler.RefreshFeeds)

	// Protected API routes
	api := r.Group("/api")
	api.Use(authMiddleware.RequireAuth())
	{
		api.GET("/feeds", feedHandler.GetFeeds)
		api.POST("/feeds", feedHandler.AddFeed)
		api.POST("/feeds/import", feedHandler.ImportOPML)
		api.DELETE("/feeds/:id", feedHandler.DeleteFeed)
		api.GET("/feeds/:id/articles", feedHandler.GetArticles)
		api.GET("/feeds/:id/debug", feedHandler.DebugFeed)
		api.GET("/feeds/unread-counts", feedHandler.GetUnreadCounts)
		api.GET("/subscription", feedHandler.GetSubscriptionInfo)
		api.GET("/account/stats", feedHandler.GetAccountStats)
		api.POST("/articles/:id/read", feedHandler.MarkRead)
		api.POST("/articles/:id/star", feedHandler.ToggleStar)
		api.POST("/feeds/refresh", feedHandler.RefreshFeeds)  // Keep for authenticated manual refresh
		
		// Payment/subscription routes - only if subscriptions are enabled
		if cfg.SubscriptionEnabled && paymentHandler != nil {
			api.GET("/stripe/config", paymentHandler.GetStripeConfig)
			api.POST("/subscription/checkout", paymentHandler.CreateCheckoutSession)
			api.POST("/subscription/portal", paymentHandler.CreateCustomerPortal)
		}
	}

	// Webhook routes (public - no auth required) - only if subscriptions are enabled
	if cfg.SubscriptionEnabled && paymentHandler != nil {
		r.POST("/webhooks/stripe", paymentHandler.WebhookHandler)
	}

	// Get port from configuration
	port := cfg.Port

	log.Fatal(r.Run(":" + port))
}
