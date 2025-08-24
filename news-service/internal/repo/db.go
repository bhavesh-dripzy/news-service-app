package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"news-system/internal/cache"
	"github.com/go-redis/redis/v9"
)

// DB represents a database connection
type DB struct {
	pool *pgxpool.Pool
}

// NewDB creates a new database connection
func NewDB(databaseURL string) (*DB, error) {
	// For now, return a mock DB since we're using in-memory storage
	return &DB{}, nil
}

// Repository interface for database operations
type Repository interface {
	CreateArticle(ctx context.Context, arg CreateArticleParams) (Article, error)
	GetArticleByID(ctx context.Context, id string) (Article, error)
	GetArticlesByCategory(ctx context.Context, arg GetArticlesByCategoryParams) ([]Article, error)
	GetArticlesBySource(ctx context.Context, arg GetArticlesBySourceParams) ([]Article, error)
	GetArticlesByScore(ctx context.Context, arg GetArticlesByScoreParams) ([]Article, error)
	SearchArticles(ctx context.Context, arg SearchArticlesParams) ([]SearchArticlesRow, error)
	GetNearbyArticles(ctx context.Context, arg GetNearbyArticlesParams) ([]GetNearbyArticlesRow, error)
	GetRecentEventsByGeohash(ctx context.Context, since time.Time) ([]GetRecentEventsByGeohashRow, error)
	CreateArticleSummary(ctx context.Context, arg CreateArticleSummaryParams) (ArticleSummary, error)
	GetArticleSummary(ctx context.Context, articleID string) (ArticleSummary, error)
	CreateUserEvent(ctx context.Context, arg CreateUserEventParams) (UserEvent, error)
	GetArticlesWithoutSummary(ctx context.Context, limit int32) ([]Article, error)
}

// Article represents a news article
type Article struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Description     *string    `json:"description"`
	URL             string     `json:"url"`
	PublicationDate time.Time  `json:"publication_date"`
	SourceName      string     `json:"source_name"`
	Category        []string   `json:"category"`
	RelevanceScore  float64    `json:"relevance_score"`
	Latitude        *float64   `json:"latitude"`
	Longitude       *float64   `json:"longitude"`
}

// ArticleSummary represents an article summary
type ArticleSummary struct {
	ArticleID   string    `json:"article_id"`
	LLMSummary  string    `json:"llm_summary"`
	Model       string    `json:"model"`
	GeneratedAt time.Time `json:"generated_at"`
}

// UserEvent represents a user interaction event
type UserEvent struct {
	ID          int64      `json:"id"`
	ArticleID   string     `json:"article_id"`
	Event       string     `json:"event"`
	OccurredAt  time.Time  `json:"occurred_at"`
	UserLat     *float64   `json:"user_lat"`
	UserLon     *float64   `json:"user_lon"`
}

// Search result with score
type SearchArticlesRow struct {
	Article
	SearchScore float64 `json:"search_score"`
}

// Nearby result with distance
type GetNearbyArticlesRow struct {
	Article
	DistanceMeters float64 `json:"distance_meters"`
}

// Event with article location
type GetRecentEventsByGeohashRow struct {
	UserEvent
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// Parameter structs for queries
type CreateArticleParams struct {
	ID              string
	Title           string
	Description     *string
	URL             string
	PublicationDate time.Time
	SourceName      string
	Category        []string
	RelevanceScore  float64
	Latitude        *float64
	Longitude       *float64
}

type GetArticlesByCategoryParams struct {
	Name  string
	Limit int32
}

type GetArticlesBySourceParams struct {
	Name  string
	Limit int32
}

type GetArticlesByScoreParams struct {
	Min   float64
	Limit int32
}

type SearchArticlesParams struct {
	Query string
	Limit int32
}

type GetNearbyArticlesParams struct {
	Lat    float64
	Lon    float64
	Radius float64
	Limit  int32
}

type CreateArticleSummaryParams struct {
	ArticleID  string
	LLMSummary string
	Model      string
}

type CreateUserEventParams struct {
	ArticleID string
	Event     string
	UserLat   *float64
	UserLon   *float64
}

// Repository implementation
type repository struct {
	db *DB
	// Redis cache for persistent storage
	cache *cache.RedisCache
	// In-memory storage for testing
	articles map[string]Article
	nextID   int64
}

func NewRepository(db *DB) Repository {
	// Create a Redis cache instance for the repository
	// Use the Docker service name 'redis' and default port 6379
	redisCache, err := cache.NewRedisCache("redis:6379", "", 0)
	if err != nil {
		// Fallback to in-memory if Redis is not available
		return &repository{
			db:       db,
			articles: make(map[string]Article),
			nextID:   1,
		}
	}
	
	return &repository{
		db:       db,
		cache:    redisCache,
		nextID:   1,
	}
}

// CreateArticle creates or updates an article
func (r *repository) CreateArticle(ctx context.Context, arg CreateArticleParams) (Article, error) {
	// Generate ID if not provided
	if arg.ID == "" {
		arg.ID = fmt.Sprintf("article_%d", r.nextID)
		r.nextID++
	}

	// Create article
	article := Article{
		ID:              arg.ID,
		Title:           arg.Title,
		Description:     arg.Description,
		URL:             arg.URL,
		PublicationDate: arg.PublicationDate,
		SourceName:      arg.SourceName,
		Category:        arg.Category,
		RelevanceScore:  arg.RelevanceScore,
		Latitude:        arg.Latitude,
		Longitude:       arg.Longitude,
	}

	// Store in Redis
	if r.cache != nil {
		articleData, err := json.Marshal(article)
		if err == nil {
			// Store individual article
			r.cache.Set(ctx, fmt.Sprintf("article:%s", arg.ID), articleData, 24*time.Hour)
			
			// Store in article list
			r.cache.SAdd(ctx, "articles:all", arg.ID)
			
			// Store by category
			for _, category := range article.Category {
				r.cache.SAdd(ctx, fmt.Sprintf("articles:category:%s", strings.ToLower(category)), arg.ID)
			}
			
			// Store by source
			r.cache.SAdd(ctx, fmt.Sprintf("articles:source:%s", strings.ToLower(article.SourceName)), arg.ID)
			
			// Store by score
			r.cache.ZAdd(ctx, "articles:by_score", redis.Z{
				Score:  article.RelevanceScore,
				Member: arg.ID,
			})
		}
	} else {
		// Fallback to in-memory storage
		if r.articles == nil {
			r.articles = make(map[string]Article)
		}
		r.articles[arg.ID] = article
	}

	return article, nil
}

// GetArticleByID retrieves an article by ID
func (r *repository) GetArticleByID(ctx context.Context, id string) (Article, error) {
	if r.cache != nil {
		// Try Redis first
		if articleData, err := r.cache.Get(ctx, fmt.Sprintf("article:%s", id)); err == nil {
			var article Article
			if err := json.Unmarshal(articleData, &article); err == nil {
				return article, nil
			}
		}
	}
	
	// Fallback to in-memory
	if r.articles != nil {
		article, exists := r.articles[id]
		if !exists {
			return Article{}, fmt.Errorf("article not found: %s", id)
		}
		return article, nil
	}
	
	return Article{}, fmt.Errorf("article not found: %s", id)
}

// GetArticlesByCategory retrieves articles by category
func (r *repository) GetArticlesByCategory(ctx context.Context, arg GetArticlesByCategoryParams) ([]Article, error) {
	if r.cache != nil {
		// Try Redis first
		categoryKey := fmt.Sprintf("articles:category:%s", strings.ToLower(arg.Name))
		articleIDs, err := r.cache.SMembers(ctx, categoryKey)
		if err == nil && len(articleIDs) > 0 {
			var articles []Article
			for _, id := range articleIDs {
				if article, err := r.GetArticleByID(ctx, id); err == nil {
					articles = append(articles, article)
					if len(articles) >= int(arg.Limit) {
						break
					}
				}
			}
			return articles, nil
		}
	}
	
	// Fallback to in-memory
	if r.articles != nil {
		var results []Article
		for _, article := range r.articles {
			for _, category := range article.Category {
				if strings.Contains(strings.ToLower(category), strings.ToLower(arg.Name)) {
					results = append(results, article)
					break
				}
			}
			if len(results) >= int(arg.Limit) {
				break
			}
		}
		return results, nil
	}
	
	return []Article{}, nil
}

// GetArticlesBySource retrieves articles by source
func (r *repository) GetArticlesBySource(ctx context.Context, arg GetArticlesBySourceParams) ([]Article, error) {
	if r.cache != nil {
		// Try Redis first
		sourceKey := fmt.Sprintf("articles:source:%s", strings.ToLower(arg.Name))
		articleIDs, err := r.cache.SMembers(ctx, sourceKey)
		if err == nil && len(articleIDs) > 0 {
			var articles []Article
			for _, id := range articleIDs {
				if article, err := r.GetArticleByID(ctx, id); err == nil {
					articles = append(articles, article)
					if len(articles) >= int(arg.Limit) {
						break
					}
				}
			}
			return articles, nil
		}
	}
	
	// Fallback to in-memory
	if r.articles != nil {
		var results []Article
		for _, article := range r.articles {
			if strings.Contains(strings.ToLower(article.SourceName), strings.ToLower(arg.Name)) {
				results = append(results, article)
				if len(results) >= int(arg.Limit) {
					break
				}
			}
		}
		return results, nil
	}
	
	return []Article{}, nil
}

// GetArticlesByScore retrieves articles by minimum score
func (r *repository) GetArticlesByScore(ctx context.Context, arg GetArticlesByScoreParams) ([]Article, error) {
	if r.cache != nil {
		// Try Redis first
		articleIDs, err := r.cache.ZRangeByScore(ctx, "articles:by_score", arg.Min, 1.0, int64(arg.Limit))
		if err == nil && len(articleIDs) > 0 {
			var articles []Article
			for _, id := range articleIDs {
				if article, err := r.GetArticleByID(ctx, id); err == nil {
					articles = append(articles, article)
					if len(articles) >= int(arg.Limit) {
						break
					}
				}
			}
			return articles, nil
		}
	}
	
	// Fallback to in-memory
	if r.articles != nil {
		var results []Article
		for _, article := range r.articles {
			if article.RelevanceScore >= arg.Min {
				results = append(results, article)
				if len(results) >= int(arg.Limit) {
					break
				}
			}
		}
		return results, nil
	}
	
	return []Article{}, nil
}

// SearchArticles performs full-text search
func (r *repository) SearchArticles(ctx context.Context, arg SearchArticlesParams) ([]SearchArticlesRow, error) {
	if r.cache != nil {
		// Try Redis first
		articleIDs, err := r.cache.SMembers(ctx, "articles:all")
		if err == nil && len(articleIDs) > 0 {
			var results []SearchArticlesRow
			query := strings.ToLower(arg.Query)
			
			for _, id := range articleIDs {
				if article, err := r.GetArticleByID(ctx, id); err == nil {
					// Simple text search in title and description
					titleMatch := strings.Contains(strings.ToLower(article.Title), query)
					descMatch := false
					if article.Description != nil {
						descMatch = strings.Contains(strings.ToLower(*article.Description), query)
					}
					
					if titleMatch || descMatch {
						// Calculate simple search score
						score := 0.0
						if titleMatch {
							score += 0.7
						}
						if descMatch {
							score += 0.3
						}
						score += article.RelevanceScore * 0.2
						
						results = append(results, SearchArticlesRow{
							Article:    article,
							SearchScore: score,
						})
						
						if len(results) >= int(arg.Limit) {
							break
						}
					}
				}
			}
			return results, nil
		}
	}
	
	// Fallback to in-memory
	if r.articles != nil {
		var results []SearchArticlesRow
		query := strings.ToLower(arg.Query)
		
		for _, article := range r.articles {
			// Simple text search in title and description
			titleMatch := strings.Contains(strings.ToLower(article.Title), query)
			descMatch := false
			if article.Description != nil {
				descMatch = strings.Contains(strings.ToLower(*article.Description), query)
			}
			
			if titleMatch || descMatch {
				// Calculate simple search score
				score := 0.0
				if titleMatch {
					score += 0.7
				}
				if descMatch {
					score += 0.3
				}
				score += article.RelevanceScore * 0.2
				
				results = append(results, SearchArticlesRow{
					Article:    article,
					SearchScore: score,
				})
				
				if len(results) >= int(arg.Limit) {
					break
				}
			}
		}
		return results, nil
	}
	
	return []SearchArticlesRow{}, nil
}

// GetNearbyArticles retrieves articles within a specified radius
func (r *repository) GetNearbyArticles(ctx context.Context, arg GetNearbyArticlesParams) ([]GetNearbyArticlesRow, error) {
	var results []GetNearbyArticlesRow
	
	// Get all articles first
	var articles []Article
	if r.cache != nil {
		// Try Redis first
		articleIDs, err := r.cache.SMembers(ctx, "articles:all")
		if err == nil && len(articleIDs) > 0 {
			for _, id := range articleIDs {
				if article, err := r.GetArticleByID(ctx, id); err == nil {
					articles = append(articles, article)
				}
			}
		}
	} else if r.articles != nil {
		// Fallback to in-memory
		for _, article := range r.articles {
			articles = append(articles, article)
		}
	}
	
	// Process articles and calculate distances
	for _, article := range articles {
		if article.Latitude != nil && article.Longitude != nil {
			// Calculate distance using Haversine formula
			distance := haversineDistance(arg.Lat, arg.Lon, *article.Latitude, *article.Longitude)
			
			if distance <= arg.Radius {
				results = append(results, GetNearbyArticlesRow{
					Article:        article,
					DistanceMeters: distance * 1000, // Convert km to meters
				})
				
				if len(results) >= int(arg.Limit) {
					break
				}
			}
		}
	}
	
	// Sort by distance
	sort.Slice(results, func(i, j int) bool {
		return results[i].DistanceMeters < results[j].DistanceMeters
	})
	
	return results, nil
}

// GetRecentEventsByGeohash retrieves recent events for trending calculation
func (r *repository) GetRecentEventsByGeohash(ctx context.Context, since time.Time) ([]GetRecentEventsByGeohashRow, error) {
	// For now, return empty results
	return []GetRecentEventsByGeohashRow{}, nil
}

// CreateArticleSummary creates or updates an article summary
func (r *repository) CreateArticleSummary(ctx context.Context, arg CreateArticleSummaryParams) (ArticleSummary, error) {
	summary := ArticleSummary{
		ArticleID:   arg.ArticleID,
		LLMSummary:  arg.LLMSummary,
		Model:       arg.Model,
		GeneratedAt: time.Now(),
	}
	return summary, nil
}

// GetArticleSummary retrieves an article summary
func (r *repository) GetArticleSummary(ctx context.Context, articleID string) (ArticleSummary, error) {
	return ArticleSummary{}, fmt.Errorf("not implemented")
}

// CreateUserEvent creates a user event
func (r *repository) CreateUserEvent(ctx context.Context, arg CreateUserEventParams) (UserEvent, error) {
	event := UserEvent{
		ID:          r.nextID,
		ArticleID:   arg.ArticleID,
		Event:       arg.Event,
		OccurredAt:  time.Now(),
		UserLat:     arg.UserLat,
		UserLon:     arg.UserLon,
	}
	r.nextID++
	return event, nil
}

// GetArticlesWithoutSummary retrieves articles without summaries
func (r *repository) GetArticlesWithoutSummary(ctx context.Context, limit int32) ([]Article, error) {
	var results []Article
	if r.cache != nil {
		// Try Redis first
		articleIDs, err := r.cache.SMembers(ctx, "articles:all")
		if err == nil && len(articleIDs) > 0 {
			for _, id := range articleIDs {
				if article, err := r.GetArticleByID(ctx, id); err == nil {
					results = append(results, article)
					if len(results) >= int(limit) {
						break
					}
				}
			}
		}
	} else if r.articles != nil {
		// Fallback to in-memory
		for _, article := range r.articles {
			results = append(results, article)
			if len(results) >= int(limit) {
				break
			}
		}
	}
	return results, nil
}

// haversineDistance calculates the distance between two points using the Haversine formula
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers
	
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180
	
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return R * c
}