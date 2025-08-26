package main

import (
	"expense-tracker/controllers"
	"expense-tracker/database"
	"expense-tracker/middleware"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	if err := database.Initialize(nil); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	// Create router
	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORS())
	router.Use(middleware.RateLimit())

	// Initialize controllers
	authController := controllers.NewAuthController()
	eventController := controllers.NewEventController()
	expenseController := controllers.NewExpenseController()
	settlementController := controllers.NewSettlementController()
	wsController := controllers.NewWebSocketController()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		if err := database.HealthCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"version":   "1.0.0",
			"database":  "connected",
			"timestamp": database.DB.NowFunc(),
		})
	})

	// Serve static files
	router.Static("/static", "./static")
	router.LoadHTMLGlob("templates/**/*.html")

	// Frontend routes (serve HTML pages)
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login_swiss.html", gin.H{
			"title": "ExpenseTracker - Login",
		})
	})

	router.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register_simple.html", gin.H{
			"title": "ExpenseTracker - Register",
		})
	})

	router.GET("/dashboard", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard_swiss.html", gin.H{
			"title": "Dashboard",
		})
	})

	router.GET("/events/:eventId", func(c *gin.Context) {
		c.HTML(http.StatusOK, "details.html", gin.H{
			"title": "Event Details",
		})
	})

	router.GET("/event-details", func(c *gin.Context) {
		c.HTML(http.StatusOK, "event_details_swiss.html", gin.H{
			"title": "Event Details",
		})
	})

	router.GET("/test-simple", func(c *gin.Context) {
		c.HTML(http.StatusOK, "test_simple.html", gin.H{
			"title": "Simple Test",
		})
	})

	router.GET("/demo", func(c *gin.Context) {
		c.HTML(http.StatusOK, "demo.html", gin.H{
			"title": "ExpenseTracker Demo - Splitwise Features",
		})
	})

	// API routes
	api := router.Group("/api/v1")

	// Public API routes (no authentication required)
	publicAPI := api.Group("/")
	{
		publicAPI.POST("/auth/register", middleware.AuthRateLimit(), authController.Register)
		publicAPI.POST("/auth/login", middleware.AuthRateLimit(), authController.Login)
		publicAPI.POST("/events/join", middleware.AuthMiddleware(), eventController.JoinEvent)
	}

	// Protected API routes (authentication required)
	protectedAPI := api.Group("/")
	protectedAPI.Use(middleware.AuthMiddleware())
	{
		// Auth endpoints
		authAPI := protectedAPI.Group("/auth")
		{
			authAPI.GET("/profile", authController.GetProfile)
			authAPI.PUT("/profile", authController.UpdateProfile)
			authAPI.POST("/change-password", authController.ChangePassword)
			authAPI.POST("/refresh", authController.RefreshToken)
		}

		// Event endpoints
		eventsAPI := protectedAPI.Group("/events")
		{
			eventsAPI.POST("/", eventController.CreateEvent)
			eventsAPI.GET("/", eventController.GetUserEvents)
			eventsAPI.GET("/:eventId", middleware.RequireEventParticipation(), eventController.GetEventDetails)
			eventsAPI.GET("/:eventId/summary", middleware.RequireEventParticipation(), eventController.GetEventSummary)
			eventsAPI.PUT("/:eventId", middleware.RequireEventAdmin(), eventController.UpdateEvent)
			eventsAPI.POST("/:eventId/contributions", middleware.RequireEventParticipation(), eventController.AddContribution)
		}

		// Expense endpoints
		expensesAPI := protectedAPI.Group("/expenses/event/:eventId")
		expensesAPI.Use(middleware.RequireEventParticipation())
		{
			expensesAPI.POST("/", expenseController.CreateExpense)
			expensesAPI.GET("/", expenseController.GetEventExpenses)
		}

		// Individual expense endpoints
		expenseAPI := protectedAPI.Group("/expenses")
		{
			expenseAPI.GET("/:id", expenseController.GetExpenseDetails)
			expenseAPI.PUT("/:id", expenseController.UpdateExpense)
			expenseAPI.DELETE("/:id", expenseController.DeleteExpense)
			expenseAPI.POST("/:id/review", expenseController.ReviewExpense)
		}

		// Expense categories
		protectedAPI.GET("/categories", expenseController.GetExpenseCategories)

		// Settlement endpoints
		settlementsAPI := protectedAPI.Group("/settlements/event/:eventId")
		settlementsAPI.Use(middleware.RequireEventParticipation())
		{
			settlementsAPI.GET("/", settlementController.GetEventSettlements)
			settlementsAPI.GET("/balances", settlementController.GetEventBalances)
			settlementsAPI.GET("/summary", settlementController.GetSettlementSummary)
			settlementsAPI.GET("/user", settlementController.GetUserSettlements)
			settlementsAPI.POST("/generate", middleware.RequireEventAdmin(), settlementController.GenerateOptimalSettlements)
			settlementsAPI.POST("/custom", settlementController.CreateCustomSettlement)
		}

		// Individual settlement endpoints
		settlementAPI := protectedAPI.Group("/settlements")
		{
			settlementAPI.GET("/:eventId", settlementController.GetSettlementDetails)
			settlementAPI.POST("/:eventId/complete", settlementController.CompleteSettlement)
		}

		// WebSocket endpoint
		wsAPI := protectedAPI.Group("/ws")
		{
			wsAPI.GET("/events/:eventId", wsController.HandleWebSocket)
		}

		// Admin endpoints (for debugging and monitoring)
		adminAPI := protectedAPI.Group("/admin")
		{
			adminAPI.GET("/stats", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"database_stats":     database.GetStats(),
					"active_connections": wsController.GetActiveConnections(),
				})
			})
		}
	}

	// Advanced Event Sourcing Demo API (Phases 4-6) - Public for demonstration
	simpleAdvancedController := controllers.NewSimpleAdvancedController()
	defer simpleAdvancedController.Stop() // Cleanup on server shutdown

	advancedAPI := api.Group("/advanced")
	{
		// Demo overview
		advancedAPI.GET("/", simpleAdvancedController.DemoOverview)

		// Phase 4: Saga Workflows
		advancedAPI.POST("/events/:eventId/saga", simpleAdvancedController.StartSagaWorkflow)
		advancedAPI.GET("/saga/:sagaId", simpleAdvancedController.GetSagaStatus)

		// Phase 5: CQRS Queries
		advancedAPI.GET("/users/:userId/events", simpleAdvancedController.ExecuteUserEventsQuery)
		advancedAPI.GET("/events/:eventId/details", simpleAdvancedController.ExecuteEventDetailsQuery)

		// Phase 6: Performance Monitoring
		advancedAPI.GET("/metrics", simpleAdvancedController.GetPerformanceMetrics)
		advancedAPI.GET("/health", simpleAdvancedController.GetSystemHealth)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting ExpenseTracker server on port %s", port)
	log.Printf("Access the application at: http://localhost:%s", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
