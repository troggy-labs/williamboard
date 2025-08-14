package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lincolngreen/williamboard/api/config"
	"github.com/lincolngreen/williamboard/api/handlers"
	"github.com/lincolngreen/williamboard/api/middleware"
	"github.com/lincolngreen/williamboard/api/models"
	"github.com/lincolngreen/williamboard/api/services"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := connectDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the schema
	if err := migrateDB(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize services
	storageService := services.NewStorageService(cfg)
	
	// Initialize handlers
	uploadHandler := handlers.NewUploadHandler(cfg, db, storageService)
	submissionHandler := handlers.NewSubmissionHandler(cfg, db)
	eventHandler := handlers.NewEventHandler(cfg, db)
	adminHandler := handlers.NewAdminHandler(cfg, db)

	// Setup router
	router := setupRouter(cfg, uploadHandler, submissionHandler, eventHandler, adminHandler, storageService)

	log.Printf("Starting %s API server on port %s", cfg.AppName, cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, router))
}

func connectDB(cfg *config.Config) (*gorm.DB, error) {
	var logLevel logger.LogLevel
	if cfg.Environment == "development" {
		logLevel = logger.Info
	} else {
		logLevel = logger.Warn
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return db, nil
}

func migrateDB(db *gorm.DB) error {
	// Create required extensions first
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		return fmt.Errorf("failed to create uuid-ossp extension: %w", err)
	}
	
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "postgis"`).Error; err != nil {
		return fmt.Errorf("failed to create postgis extension: %w", err)
	}
	
	// Now run AutoMigrate
	return db.AutoMigrate(
		&models.Submission{},
		&models.Flyer{},
		&models.Venue{},
		&models.EventCandidate{},
		&models.Event{},
		&models.DedupeLink{},
		&models.AuditLog{},
		&models.Flag{},
	)
}

func setupRouter(
	cfg *config.Config,
	uploadHandler *handlers.UploadHandler,
	submissionHandler *handlers.SubmissionHandler,
	eventHandler *handlers.EventHandler,
	adminHandler *handlers.AdminHandler,
	storageService *services.StorageService,
) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Create template with custom functions
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"ge": func(a, b float64) bool {
			return a >= b
		},
		"gt": func(a, b float64) bool {
			return a > b
		},
		"printf": fmt.Sprintf,
	}).ParseGlob("api/templates/*"))
	router.SetHTMLTemplate(tmpl)

	// Middleware
	router.Use(middleware.CORS())
	router.Use(middleware.Logger())
	router.Use(middleware.ErrorHandler())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"app":    cfg.AppName,
		})
	})

	// Static file serving
	router.Static("/files", storageService.GetUploadDir())

	// API routes
	v1 := router.Group("/v1")
	{
		// Upload endpoints
		uploads := v1.Group("/uploads")
		{
			uploads.POST("/signed-url", uploadHandler.GetSignedURL)
			uploads.PUT("/:id", uploadHandler.UploadFile)
		}

		// Submission endpoints (for checking results after upload)
		submissions := v1.Group("/submissions")
		{
			submissions.GET("/:id/status", submissionHandler.GetStatus)
		}

		// Event endpoints
		events := v1.Group("/events")
		{
			events.GET("", eventHandler.List)
			events.GET("/:id", eventHandler.Get)
			events.GET("/:id/ics", eventHandler.GetICS)
			events.POST("/:id/unpublish", eventHandler.Unpublish)
		}
	}

	// Admin routes
	admin := router.Group("/admin")
	{
		handlers.RegisterAdminRoutes(admin, adminHandler)
	}

	return router
}