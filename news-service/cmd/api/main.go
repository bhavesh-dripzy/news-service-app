package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"news-system/internal/cache"
	"news-system/internal/config"
	httphandler "news-system/internal/http"
	"news-system/internal/ingest"
	"news-system/internal/repo"
	"news-system/internal/services/llm"
	"news-system/internal/services/news"
	"news-system/internal/services/trending"
)

func main() {
	// Parse command line flags
	var (
		ingestData = flag.Bool("ingest", false, "Load sample data into the database")
		port       = flag.String("port", "8080", "Port to run the server on")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, err := repo.NewDB(cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Note: db.Close() is not needed for mock DB

	// Initialize repository
	repository := repo.NewRepository(db)

	// Initialize Redis cache
	redisCache, err := cache.NewRedisCache(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	// Initialize LLM client
	llmClient, err := llm.NewOpenAIClient(cfg.OpenAI.APIKey, cfg.OpenAI.Model)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	// Initialize services
	newsService := news.NewNewsService(repository, redisCache, llmClient)
	trendingScorer := trending.NewTrendingScorer(repository, redisCache)

	// Initialize ingestion loader
	loader := ingest.NewLoader(repository)

	// If ingest flag is set, load sample data and exit
	if *ingestData {
		log.Println("Loading sample data...")
		if err := loader.GenerateSampleData(ctx); err != nil {
			log.Fatalf("Failed to load sample data: %v", err)
		}
		log.Println("Sample data loaded successfully!")
		return
	}

	// Start trending scorer
	trendingScorer.Start(ctx, cfg.Trending.WorkerInterval)
	defer trendingScorer.Stop()

	// Simulate some user events for trending
	go func() {
		time.Sleep(2 * time.Second) // Wait for services to be ready
		if err := trendingScorer.SimulateUserEvents(ctx); err != nil {
			log.Printf("Failed to simulate user events: %v", err)
		}
	}()

	// Initialize HTTP router
	router := httphandler.NewRouter()
	
	// Register routes
	newsHandler := httphandler.NewNewsHandler(newsService)
	router.RegisterNewsRoutes(newsHandler)
	router.RegisterHealthRoutes()
	router.RegisterMetricsRoutes()

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + *port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown server gracefully
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
