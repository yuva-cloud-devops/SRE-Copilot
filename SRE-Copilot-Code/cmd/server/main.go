package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"sre-copilot/internal/config"
	"sre-copilot/internal/database"
	"sre-copilot/internal/handlers"
	"sre-copilot/internal/repository"
	"sre-copilot/internal/services"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting SRE Copilot Monolith [Env: %s, AI Provider: %s]...", cfg.Environment, cfg.AIProvider)

	// 1. Initialize Database & Repository
	var repo repository.IncidentRepository
	var knowledgeRepo repository.KnowledgeRepository

	db, err := database.InitDB(cfg)
	if err == nil && db != nil {
		log.Println("✓ Connected to PostgreSQL database successfully. Using PostgresRepository.")
		pgRepo := repository.NewPostgresRepository(db)
		repo = pgRepo
		knowledgeRepo = pgRepo
	} else {
		log.Printf("⚠ PostgreSQL unavailable (%v). Falling back to MemoryRepository with seed data.", err)
		memRepo := repository.NewMemoryRepository()
		repo = memRepo
		knowledgeRepo = memRepo
	}

	// 2. Initialize Core Services (Pure Go in-memory channels for SSE, no Redis required)
	broadcaster := services.NewSSEBroadcaster()
	correlator := services.NewAlertCorrelator(repo, broadcaster)
	incidentSvc := services.NewIncidentService(repo, broadcaster)
	knowledgeSvc := services.NewKnowledgeService(knowledgeRepo)
	aiSvc := services.NewAICopilotService(cfg)

	// 3. Initialize Handlers
	webHandler := handlers.NewWebHandler(incidentSvc, knowledgeSvc, aiSvc)
	apiHandler := handlers.NewAPIHandler(incidentSvc, knowledgeSvc, aiSvc, broadcaster)
	webhookHandler := handlers.NewWebhookHandler(correlator)

	// 4. Configure Gin Router
	router := gin.Default()

	// Register custom template helper functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}
	router.SetFuncMap(funcMap)

	// Load HTML templates
	router.LoadHTMLGlob("templates/**/*.html")

	// Static files
	router.Static("/static", "./static")

	// Web UI Navigation Routes (Go HTML Templates)
	router.GET("/", webHandler.DashboardPage)
	router.GET("/incidents", webHandler.IncidentsPage)
	router.GET("/incidents/:id", webHandler.IncidentDetailPage)
	router.GET("/rca", webHandler.RCAListPage)
	router.GET("/incidents/:id/rca", webHandler.RCAPage)
	router.GET("/knowledge", webHandler.KnowledgePage)

	// API & Real-Time SSE Routes
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/stream", apiHandler.SSEGlobalStream)
		apiGroup.POST("/incidents/:id/status", apiHandler.UpdateIncidentStatus)
		apiGroup.POST("/incidents/:id/comment", apiHandler.AddComment)
		apiGroup.POST("/incidents/:id/ai/triage", apiHandler.AITriage)
		apiGroup.POST("/incidents/:id/ai/chat", apiHandler.AIChatStream)

		// Ingestion Webhooks: Google Cloud Monitoring & Cloud Logging
		apiGroup.POST("/webhooks/gcp/monitoring", webhookHandler.GCPMonitoringWebhook)
		apiGroup.POST("/webhooks/gcp/logging", webhookHandler.GCPLoggingWebhook)
	}

	// Liveness & Readiness probe
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":       "healthy",
			"app":          "SRE Copilot Monolith",
			"ai_provider":  cfg.AIProvider,
			"gcp_project":  cfg.GCPProjectID,
			"db_connected": db != nil,
		})
	})

	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("⚡ SRE Copilot running at http://localhost%s", serverAddr)
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}
