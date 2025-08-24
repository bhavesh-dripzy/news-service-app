package http

import (
	"net/http"
	"time"

	"news-system/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Router struct {
	chi.Router
}

func NewRouter() *Router {
	r := chi.NewRouter()
	
	// Use chi middleware with aliases to avoid conflicts
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	
	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	
	// Custom middleware
	r.Use(middleware.RateLimit)
	r.Use(middleware.Logging)
	
	return &Router{r}
}

// RegisterNewsRoutes registers news-related routes
func (r *Router) RegisterNewsRoutes(newsHandler *NewsHandler) {
	newsHandler.RegisterRoutes(r)
}

// RegisterHealthRoutes registers health check routes
func (r *Router) RegisterHealthRoutes() {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
	})
	
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
	})
}

// RegisterMetricsRoutes registers metrics routes
func (r *Router) RegisterMetricsRoutes() {
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement Prometheus metrics endpoint
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# Metrics endpoint - implement Prometheus metrics here\n"))
	})
}
