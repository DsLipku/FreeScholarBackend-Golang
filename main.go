package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"freescholar-backend/api/routes"
	"freescholar-backend/config"
	"freescholar-backend/internal/models"
	"freescholar-backend/pkg/elasticsearch"
	"freescholar-backend/pkg/mysql"
	"freescholar-backend/pkg/redis"

	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("./config.yaml", "./secrets.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up MySQL connection with GORM
	db, err := mysql.NewClient(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	
	// Don't close until server shutdown
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}
	defer sqlDB.Close()

	// Auto migrate database schema
	log.Println("Migrating database schema...")
	if err := migrateDB(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Set up Redis connection
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Set up Elasticsearch connection
	esClient, err := elasticsearch.NewClient(cfg.ES)
	if err != nil {
		log.Fatalf("Failed to connect to Elasticsearch: %v", err)
	}

	// Set up Gin router with routes
	router := routes.SetupRouter(cfg, db, redisClient, esClient)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start the server in a goroutine
	go func() {
		fmt.Printf("Starting server at %s:%d\n", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server exiting")
}

// migrateDB performs database migrations using GORM
func migrateDB(db *gorm.DB) error {
	// Add all models that need to be migrated
	return db.AutoMigrate(
		&models.User{},
		&models.Publication{},
		&models.Author{},
		&models.PublicationAuthor{},
		&models.Relation{},
		&models.Message{},
		&models.ScholarProfile{},
		&models.SearchHistory{},
		&models.File{},
		&models.Serialization{},
	)
}