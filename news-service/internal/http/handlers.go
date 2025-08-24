package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"news-system/internal/services/news"
	"github.com/go-chi/chi/v5"
)

// NewsHandler handles news-related HTTP requests
type NewsHandler struct {
	newsService *news.NewsService
}

// NewNewsHandler creates a new NewsHandler
func NewNewsHandler(newsService *news.NewsService) *NewsHandler {
	return &NewsHandler{newsService: newsService}
}

// RegisterRoutes registers all news routes
func (h *NewsHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/news", func(r chi.Router) {
		r.Post("/query", h.Query)
		r.Get("/query", h.Query)
		r.Get("/trending", h.Trending)
	})
}

// Query handles unified news queries
func (h *NewsHandler) Query(w http.ResponseWriter, r *http.Request) {
	var req news.QueryRequest

	// Handle both GET and POST requests
	if r.Method == "GET" {
		// Parse query parameters
		req.Query = r.URL.Query().Get("query")
		if req.Query == "" {
			http.Error(w, "query parameter is required", http.StatusBadRequest)
			return
		}

		// Parse optional parameters
		if latStr := r.URL.Query().Get("lat"); latStr != "" {
			if lat, err := strconv.ParseFloat(latStr, 64); err == nil && lat >= -90 && lat <= 90 {
				req.Lat = &lat
			} else {
				http.Error(w, "invalid latitude value", http.StatusBadRequest)
				return
			}
		}

		if lonStr := r.URL.Query().Get("lon"); lonStr != "" {
			if lon, err := strconv.ParseFloat(lonStr, 64); err == nil && lon >= -180 && lon <= 180 {
				req.Lon = &lon
			} else {
				http.Error(w, "invalid longitude value", http.StatusBadRequest)
				return
			}
		}

		if radiusStr := r.URL.Query().Get("radius"); radiusStr != "" {
			if radius, err := strconv.ParseFloat(radiusStr, 64); err == nil && radius > 0 && radius <= 200 {
				req.Radius = &radius
			} else {
				http.Error(w, "invalid radius value (must be 0.1-200 km)", http.StatusBadRequest)
				return
			}
		}

		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 50 {
				req.Limit = limit
			} else {
				http.Error(w, "invalid limit value (must be 1-50)", http.StatusBadRequest)
				return
			}
		}
	} else {
		// Parse JSON body for POST requests
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
	}

	// Validate request
	if req.Query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	// Set default limit
	if req.Limit <= 0 {
		req.Limit = 5
	}

	// Process the query
	response, err := h.newsService.Query(r.Context(), req)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("Error processing query: %v\n", err)
		http.Error(w, fmt.Sprintf("Failed to process query: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Trending handles the bonus trending news endpoint
func (h *NewsHandler) Trending(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")
	limitStr := r.URL.Query().Get("limit")
	
	if latStr == "" || lonStr == "" {
		http.Error(w, "latitude and longitude are required", http.StatusBadRequest)
		return
	}
	
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		http.Error(w, "invalid latitude", http.StatusBadRequest)
		return
	}
	
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil || lon < -180 || lon > 180 {
		http.Error(w, "invalid longitude", http.StatusBadRequest)
		return
	}
	
	limit := 5 // Default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}
	
	// Create a trending query request
	req := news.QueryRequest{
		Query:  "trending news near me",
		Lat:    &lat,
		Lon:    &lon,
		Radius: float64Ptr(50.0), // 50km radius
		Limit:  limit,
	}
	
	// Process the trending query
	response, err := h.newsService.Query(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Helper function for creating float64 pointers
func float64Ptr(f float64) *float64 {
	return &f
}
