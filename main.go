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
	
	// Validate environment configuration
	if err := config.ValidateEnvironmentConfig(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}
	
	// Warn about potentially unhandled environment variables
	config.WarnAboutUnhandledEnvVars()

	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize rate limiter and scheduler for DDoS prevention
	rateLimiter := services.NewDomainRateLimiter(services.RateLimiterConfig{
		RequestsPerMinute: cfg.RateLimitRequestsPerMinute,
		BurstSize:         cfg.RateLimitBurstSize,
	})

	// Initialize services
	feedService := services.NewFeedService(db, rateLimiter)
	subscriptionService := services.NewSubscriptionService(db)
	authService := auth.NewAuthService(db)
	sessionManager := auth.NewSessionManager(db)
	csrfManager := auth.NewCSRFManager()

	// Initialize rate limiters for auth and API endpoints
	// Auth: 10 requests per second with burst of 20
	authRateLimiter := auth.NewRateLimiter(10, 20)
	// API: 30 requests per second with burst of 50
	apiRateLimiter := auth.NewRateLimiter(30, 50)

	// Initialize feed scheduler for staggered updates
	feedScheduler := services.NewFeedScheduler(feedService, rateLimiter, services.SchedulerConfig{
		UpdateWindow:    cfg.SchedulerUpdateWindow,
		MinInterval:     cfg.SchedulerMinInterval,
		MaxConcurrent:   cfg.SchedulerMaxConcurrent,
		CleanupInterval: cfg.SchedulerCleanupInterval,
	})

	// Start the feed scheduler for automatic staggered updates
	if err := feedScheduler.Start(); err != nil {
		log.Printf("Warning: Failed to start feed scheduler: %v", err)
	}

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
	feedHandler := handlers.NewFeedHandler(feedService, subscriptionService, feedScheduler)
	authHandler := handlers.NewAuthHandler(authService, sessionManager, csrfManager)
	adminHandler := handlers.NewAdminHandler(subscriptionService)
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
		// SECURITY: Never cache authentication endpoints
		if strings.HasPrefix(c.Request.URL.Path, "/auth/") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		// Cache static assets - shorter cache for development
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "public, max-age=60")
			c.Header("Vary", "Accept-Encoding")
		}
		// SECURITY: Never cache sensitive API endpoints containing user data
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			path := c.Request.URL.Path
			// Never cache user-specific or sensitive endpoints
			if strings.Contains(path, "/subscription") || strings.Contains(path, "/account") || 
			   strings.Contains(path, "/admin") || strings.Contains(path, "unread-counts") {
				c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
				c.Header("Pragma", "no-cache") 
				c.Header("Expires", "0")
			} else {
				// Cache other API responses for 60 seconds
				c.Header("Cache-Control", "private, max-age=60")
			}
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
	r.GET("/account", authMiddleware.RequireAuthPage(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "account.html", gin.H{
			"title": "Account Management - GoRead2",
		})
	})

	r.GET("/subscription", authMiddleware.RequireAuthPage(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "account.html", gin.H{
			"title": "Subscription Management - GoRead2",
		})
	})

	// Subscription success/cancel pages (public) - only if subscriptions are enabled
	if cfg.SubscriptionEnabled && paymentHandler != nil {
		r.GET("/subscription/success", paymentHandler.SubscriptionSuccess)
		r.GET("/subscription/cancel", paymentHandler.SubscriptionCancel)
	}

	// Auth routes (public)
	authRoutes := r.Group("/auth")
	authRoutes.Use(auth.RateLimitMiddleware(authRateLimiter)) // Rate limiting for auth endpoints
	{
		authRoutes.GET("/login", authHandler.Login)
		authRoutes.GET("/callback", authHandler.Callback)
		authRoutes.POST("/logout", authHandler.Logout)
		authRoutes.GET("/me", authMiddleware.OptionalAuth(), authHandler.Me)
	}

	// Cron endpoint - requires X-Appengine-Cron header in production or admin auth locally
	cronRoutes := r.Group("/cron")
	// In non-App Engine environments, require admin authentication
	if os.Getenv("GAE_ENV") != "standard" {
		cronRoutes.Use(authMiddleware.OptionalAuth()) // Allow admin to authenticate
	}
	{
		cronRoutes.GET("/refresh-feeds", feedHandler.RefreshFeeds)
		cronRoutes.POST("/refresh-feeds", feedHandler.RefreshFeeds)
	}

	// Protected API routes
	api := r.Group("/api")
	api.Use(auth.RateLimitMiddleware(apiRateLimiter)) // Rate limiting for API endpoints
	api.Use(authMiddleware.RequireAuth())
	api.Use(authMiddleware.CSRFMiddleware(csrfManager)) // CSRF protection for state-changing operations
	{
		api.GET("/feeds", feedHandler.GetFeeds)
		api.POST("/feeds", feedHandler.AddFeed)
		api.POST("/feeds/import", feedHandler.ImportOPML)
		api.DELETE("/feeds/:id", feedHandler.DeleteFeed)
		api.GET("/feeds/:id/articles", feedHandler.GetArticles)
		api.GET("/feeds/unread-counts", feedHandler.GetUnreadCounts)
		api.GET("/subscription", feedHandler.GetSubscriptionInfo)
		api.GET("/account/stats", feedHandler.GetAccountStats)
		api.PUT("/account/max-articles", feedHandler.UpdateMaxArticlesOnFeedAdd)
		api.POST("/articles/:id/read", feedHandler.MarkRead)
		api.POST("/articles/:id/star", feedHandler.ToggleStar)
		api.POST("/feeds/refresh", feedHandler.RefreshFeeds) // Keep for authenticated manual refresh

		// Payment/subscription routes - only if subscriptions are enabled
		if cfg.SubscriptionEnabled && paymentHandler != nil {
			api.GET("/stripe/config", paymentHandler.GetStripeConfig)
			api.POST("/subscription/checkout", paymentHandler.CreateCheckoutSession)
			api.POST("/subscription/portal", paymentHandler.CreateCustomerPortal)
		}
	}

	// Debug routes - require admin privileges
	debug := r.Group("/api/debug")
	debug.Use(authMiddleware.RequireAdmin())
	{
		debug.GET("/feeds/:id", feedHandler.DebugFeed)
		debug.GET("/article", feedHandler.DebugArticleByURL)
		debug.GET("/subscriptions", feedHandler.DebugAllSubscriptions)
	}

	// Admin routes - require admin privileges
	admin := r.Group("/admin")
	admin.Use(authMiddleware.RequireAdmin())
	admin.Use(authMiddleware.CSRFMiddleware(csrfManager)) // CSRF protection for admin operations
	{
		admin.GET("/users", adminHandler.ListUsers)
		admin.GET("/users/:email", adminHandler.GetUserInfo)
		admin.POST("/users/:email/admin", adminHandler.SetAdminStatus)
		admin.POST("/users/:email/free-months", adminHandler.GrantFreeMonths)
	}

	// Webhook routes (public - no auth required) - only if subscriptions are enabled
	if cfg.SubscriptionEnabled && paymentHandler != nil {
		r.POST("/webhooks/stripe", paymentHandler.WebhookHandler)
	}

	// Initialize admin users from environment configuration
	if err := authService.InitializeAdminUsers(); err != nil {
		log.Printf("Warning: Failed to initialize admin users: %v", err)
	}

	// Get port from configuration
	port := cfg.Port

	log.Fatal(r.Run(":" + port))
}
