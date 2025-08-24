package news

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"news-system/internal/cache"
	"news-system/internal/repo"
	"news-system/internal/services/llm"
)

// NewsService handles news retrieval and processing
type NewsService struct {
	repo repo.Repository
	cache *cache.RedisCache
	llm   llm.LLMClient
}

// NewNewsService creates a new NewsService
func NewNewsService(repo repo.Repository, cache *cache.RedisCache, llm llm.LLMClient) *NewsService {
	return &NewsService{
		repo:  repo,
		cache: cache,
		llm:   llm,
	}
}

// QueryRequest represents a unified news query request
type QueryRequest struct {
	Query    string   `json:"query" validate:"required,min=1,max=500"`
	Lat      *float64 `json:"lat,omitempty" validate:"omitempty,min=-90,max=90"`
	Lon      *float64 `json:"lon,omitempty" validate:"omitempty,min=-180,max=180"`
	Radius   *float64 `json:"radius_km,omitempty" validate:"omitempty,min=0.1,max=200"`
	Limit    int      `json:"limit" validate:"min=1,max=50"`
}

// QueryResponse represents the unified response format
type QueryResponse struct {
	Articles []ArticleDTO `json:"articles"`
	Meta     MetaInfo     `json:"meta"`
}

// MetaInfo represents metadata about the response
type MetaInfo struct {
	Total       int         `json:"total"`
	Query       *QueryInfo  `json:"query,omitempty"`
	Intent      string      `json:"intent"`
	Entities    []string    `json:"entities"`
	Strategy    string      `json:"strategy"`
}

// QueryInfo represents information about the query
type QueryInfo struct {
	Endpoint string                 `json:"endpoint"`
	Params   map[string]interface{} `json:"params"`
}

// ArticleDTO represents the article data returned to clients
type ArticleDTO struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Description     *string    `json:"description"`
	URL             string     `json:"url"`
	PublicationDate time.Time  `json:"publication_date"`
	SourceName      string     `json:"source_name"`
	Category        []string   `json:"category"`
	RelevanceScore  float64    `json:"relevance_score"`
	LLMSummary      *string    `json:"llm_summary,omitempty"`
	Latitude        *float64   `json:"latitude,omitempty"`
	Longitude       *float64   `json:"longitude,omitempty"`
	DistanceMeters  *float64   `json:"distance_meters,omitempty"`
	SearchScore     *float64   `json:"search_score,omitempty"`
}

// Query processes a unified news query using LLM to determine intent and route to appropriate strategy
func (s *NewsService) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
	// Set default limit if not provided
	if req.Limit <= 0 {
		req.Limit = 5
	}

	// Use LLM to extract entities, concepts, and determine intent
	extraction, err := s.llm.Extract(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to extract query intent: %w", err)
	}

	// Determine the appropriate data retrieval strategy
	strategy := s.determineStrategy(extraction, req)

	// Retrieve articles based on the determined strategy
	var articles []ArticleDTO
	var err2 error

	switch strategy {
	case "category":
		articles, err2 = s.getArticlesByCategory(ctx, extraction, req)
	case "source":
		articles, err2 = s.getArticlesBySource(ctx, extraction, req)
	case "score":
		articles, err2 = s.getArticlesByScore(ctx, extraction, req)
	case "search":
		articles, err2 = s.searchArticles(ctx, extraction, req)
	case "nearby":
		articles, err2 = s.getNearbyArticles(ctx, extraction, req)
	default:
		// Default to search if intent is unclear
		articles, err2 = s.searchArticles(ctx, extraction, req)
		strategy = "search"
	}

	if err2 != nil {
		return nil, fmt.Errorf("failed to retrieve articles: %w", err2)
	}

	// Enrich articles with LLM summaries
	articles = s.enrichArticles(ctx, articles)

	// Rank articles based on strategy
	articles = s.rankArticles(articles, strategy, req)

	// Limit results
	if len(articles) > req.Limit {
		articles = articles[:req.Limit]
	}

	// Build response
	response := &QueryResponse{
		Articles: articles,
		Meta: MetaInfo{
			Total:    len(articles),
			Intent:   s.getBestIntent(extraction),
			Entities: s.getAllEntities(extraction),
			Strategy: strategy,
			Query: &QueryInfo{
				Endpoint: "query",
				Params: map[string]interface{}{
					"query":  req.Query,
					"lat":    req.Lat,
					"lon":    req.Lon,
					"radius": req.Radius,
					"limit":  req.Limit,
				},
			},
		},
	}

	return response, nil
}

// determineStrategy determines the best data retrieval strategy based on LLM extraction and request
func (s *NewsService) determineStrategy(extraction *llm.Extraction, req QueryRequest) string {
	// Check for explicit location-based queries
	if req.Lat != nil && req.Lon != nil {
		return "nearby"
	}

	// Check LLM intent
	bestIntent := s.getBestIntent(extraction)
	switch strings.ToLower(bestIntent) {
	case "category", "topic":
		return "category"
	case "source", "publisher":
		return "source"
	case "score", "relevance":
		return "score"
	case "nearby", "location", "local":
		return "nearby"
	case "search", "query":
		return "search"
	default:
		// Analyze entities to determine strategy
		allEntities := s.getAllEntities(extraction)
		if s.hasSourceEntities(allEntities) {
			return "source"
		}
		if s.hasCategoryEntities(allEntities) {
			return "category"
		}
		return "search"
	}
}

// getBestIntent gets the highest confidence intent
func (s *NewsService) getBestIntent(extraction *llm.Extraction) string {
	if len(extraction.Intent) == 0 {
		return "search"
	}
	
	var bestIntent *llm.Intent
	for i := range extraction.Intent {
		if bestIntent == nil || extraction.Intent[i].Confidence > bestIntent.Confidence {
			bestIntent = &extraction.Intent[i]
		}
	}
	
	if bestIntent != nil {
		return bestIntent.Type
	}
	return "search"
}

// getAllEntities combines all entity types into a single slice
func (s *NewsService) getAllEntities(extraction *llm.Extraction) []string {
	var entities []string
	entities = append(entities, extraction.Entities.People...)
	entities = append(entities, extraction.Entities.Organizations...)
	entities = append(entities, extraction.Entities.Locations...)
	entities = append(entities, extraction.Concepts...)
	entities = append(entities, extraction.Categories...)
	entities = append(entities, extraction.SourceNames...)
	return entities
}

// hasSourceEntities checks if entities contain known news sources
func (s *NewsService) hasSourceEntities(entities []string) bool {
	sources := []string{"new york times", "reuters", "bbc", "cnn", "dw", "technews", "globalnews", "financedaily"}
	for _, entity := range entities {
		for _, source := range sources {
			if strings.Contains(strings.ToLower(entity), source) {
				return true
			}
		}
	}
	return false
}

// hasCategoryEntities checks if entities contain known news categories
func (s *NewsService) hasCategoryEntities(entities []string) bool {
	categories := []string{"technology", "business", "sports", "health", "science", "environment", "politics", "entertainment"}
	for _, entity := range entities {
		for _, category := range categories {
			if strings.Contains(strings.ToLower(entity), category) {
				return true
			}
		}
	}
	return false
}

// getArticlesByCategory retrieves articles by category
func (s *NewsService) getArticlesByCategory(ctx context.Context, extraction *llm.Extraction, req QueryRequest) ([]ArticleDTO, error) {
	// Extract category from entities or use a default
	category := "Technology" // Default
	for _, cat := range extraction.Categories {
		if s.isCategory(cat) {
			category = cat
			break
		}
	}

	// Get articles from repository
	articles, err := s.repo.GetArticlesByCategory(ctx, repo.GetArticlesByCategoryParams{
		Name:  category,
		Limit: int32(req.Limit),
	})
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	return s.convertToDTOs(articles), nil
}

// getArticlesBySource retrieves articles by source
func (s *NewsService) getArticlesBySource(ctx context.Context, extraction *llm.Extraction, req QueryRequest) ([]ArticleDTO, error) {
	// Extract source from entities
	source := "TechNews" // Default
	for _, src := range extraction.SourceNames {
		if s.isSource(src) {
			source = src
			break
		}
	}

	// Get articles from repository
	articles, err := s.repo.GetArticlesBySource(ctx, repo.GetArticlesBySourceParams{
		Name:  source,
		Limit: int32(req.Limit),
	})
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	return s.convertToDTOs(articles), nil
}

// getArticlesByScore retrieves articles by relevance score
func (s *NewsService) getArticlesByScore(ctx context.Context, extraction *llm.Extraction, req QueryRequest) ([]ArticleDTO, error) {
	// Use a default threshold for high-quality articles
	minScore := 0.8 // Default to 0.8 for high-quality articles
	
	// Try to extract score threshold from the query
	queryLower := strings.ToLower(req.Query)
	if strings.Contains(queryLower, "above") || strings.Contains(queryLower, "threshold") {
		// Look for numbers in the query
		re := regexp.MustCompile(`(\d+\.?\d*)`)
		matches := re.FindStringSubmatch(queryLower)
		if len(matches) > 1 {
			if score, err := strconv.ParseFloat(matches[1], 64); err == nil && score >= 0 && score <= 1 {
				minScore = score
			}
		}
	}

	// Get articles from repository
	articles, err := s.repo.GetArticlesByScore(ctx, repo.GetArticlesByScoreParams{
		Min:   minScore,
		Limit: int32(req.Limit),
	})
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	return s.convertToDTOs(articles), nil
}

// searchArticles performs full-text search
func (s *NewsService) searchArticles(ctx context.Context, extraction *llm.Extraction, req QueryRequest) ([]ArticleDTO, error) {
	// Use the original query for search
	query := req.Query

	// Get articles from repository
	articles, err := s.repo.SearchArticles(ctx, repo.SearchArticlesParams{
		Query: query,
		Limit: int32(req.Limit),
	})
	if err != nil {
		return nil, err
	}

	// Convert to DTOs with search scores
	dtos := make([]ArticleDTO, len(articles))
	for i, article := range articles {
		dto := s.convertToDTO(article.Article)
		dto.SearchScore = &article.SearchScore
		dtos[i] = dto
	}

	return dtos, nil
}

// getNearbyArticles retrieves articles within a specified radius
func (s *NewsService) getNearbyArticles(ctx context.Context, extraction *llm.Extraction, req QueryRequest) ([]ArticleDTO, error) {
	// Check if we have coordinates
	if req.Lat == nil || req.Lon == nil {
		// Try to extract coordinates from the query if available
		if len(extraction.Entities.Locations) > 0 {
			// For now, use a default location if coordinates aren't provided
			// In a real implementation, you'd geocode the location names
			defaultLat := 37.7749 // San Francisco
			defaultLon := -122.4194
			req.Lat = &defaultLat
			req.Lon = &defaultLon
		} else {
			return nil, fmt.Errorf("latitude and longitude are required for nearby search")
		}
	}

	radius := 10.0 // Default 10km
	if req.Radius != nil {
		radius = *req.Radius
	}

	// Get articles from repository
	articles, err := s.repo.GetNearbyArticles(ctx, repo.GetNearbyArticlesParams{
		Lat:    *req.Lat,
		Lon:    *req.Lon,
		Radius:  radius,
		Limit:   int32(req.Limit),
	})
	if err != nil {
		return nil, err
	}

	// Convert to DTOs with distance information
	dtos := make([]ArticleDTO, len(articles))
	for i, article := range articles {
		dto := s.convertToDTO(article.Article)
		dto.DistanceMeters = &article.DistanceMeters
		dtos[i] = dto
	}

	return dtos, nil
}

// enrichArticles enriches articles with LLM-generated summaries
func (s *NewsService) enrichArticles(ctx context.Context, articles []ArticleDTO) []ArticleDTO {
	// Process articles concurrently
	type result struct {
		index int
		summary string
		err    error
	}

	results := make(chan result, len(articles))
	
	for i, article := range articles {
		go func(idx int, art ArticleDTO) {
			description := ""
			if art.Description != nil {
				description = *art.Description
			}
			summary, err := s.llm.Summarize(ctx, art.Title, description, art.SourceName, art.PublicationDate.Format(time.RFC3339))
			results <- result{index: idx, summary: summary, err: err}
		}(i, article)
	}

	// Collect results
	summaries := make([]string, len(articles))
	for i := 0; i < len(articles); i++ {
		res := <-results
		if res.err == nil {
			summaries[res.index] = res.summary
		}
	}

	// Apply summaries
	for i := range articles {
		if summaries[i] != "" {
			articles[i].LLMSummary = &summaries[i]
		}
	}

	return articles
}

// rankArticles ranks articles based on the strategy used
func (s *NewsService) rankArticles(articles []ArticleDTO, strategy string, req QueryRequest) []ArticleDTO {
	switch strategy {
	case "category", "source":
		// Rank by publication date (most recent first)
		sort.Slice(articles, func(i, j int) bool {
			return articles[i].PublicationDate.After(articles[j].PublicationDate)
		})
	case "score":
		// Rank by relevance score (highest first)
		sort.Slice(articles, func(i, j int) bool {
			return articles[i].RelevanceScore > articles[j].RelevanceScore
		})
	case "search":
		// Rank by search score if available, otherwise by relevance score
		sort.Slice(articles, func(i, j int) bool {
			if articles[i].SearchScore != nil && articles[j].SearchScore != nil {
				return *articles[i].SearchScore > *articles[j].SearchScore
			}
			return articles[i].RelevanceScore > articles[j].RelevanceScore
		})
	case "nearby":
		// Rank by distance (closest first)
		sort.Slice(articles, func(i, j int) bool {
			if articles[i].DistanceMeters != nil && articles[j].DistanceMeters != nil {
				return *articles[i].DistanceMeters < *articles[j].DistanceMeters
			}
			return false
		})
	}

	return articles
}

// Helper functions
func (s *NewsService) isCategory(entity string) bool {
	categories := []string{"technology", "business", "sports", "health", "science", "environment", "politics", "entertainment"}
	for _, cat := range categories {
		if strings.Contains(strings.ToLower(entity), cat) {
			return true
		}
	}
	return false
}

func (s *NewsService) isSource(entity string) bool {
	sources := []string{"new york times", "reuters", "bbc", "cnn", "dw", "technews", "globalnews", "financedaily"}
	for _, src := range sources {
		if strings.Contains(strings.ToLower(entity), src) {
			return true
		}
	}
	return false
}

func (s *NewsService) convertToDTOs(articles []repo.Article) []ArticleDTO {
	dtos := make([]ArticleDTO, len(articles))
	for i, article := range articles {
		dtos[i] = s.convertToDTO(article)
	}
	return dtos
}

func (s *NewsService) convertToDTO(article repo.Article) ArticleDTO {
	return ArticleDTO{
		ID:              article.ID,
		Title:           article.Title,
		Description:     article.Description,
		URL:             article.URL,
		PublicationDate: article.PublicationDate,
		SourceName:      article.SourceName,
		Category:        article.Category,
		RelevanceScore:  article.RelevanceScore,
		Latitude:        article.Latitude,
		Longitude:       article.Longitude,
	}
}
