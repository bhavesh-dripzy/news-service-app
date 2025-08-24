package trending

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"news-system/internal/cache"
	"news-system/internal/repo"

	"github.com/go-redis/redis/v9"
	"github.com/rs/zerolog/log"
)

type TrendingScorer struct {
	repo   repo.Repository
	cache  *cache.RedisCache
	ticker *time.Ticker
	done   chan bool
}

type TrendingScore struct {
	ArticleID string  `json:"article_id"`
	Score     float64 `json:"score"`
}

type TrendingMeta struct {
	LastComputedAt time.Time `json:"last_computed_at"`
	EventCount     int       `json:"event_count"`
	TileCount      int       `json:"tile_count"`
}

func NewTrendingScorer(repo repo.Repository, cache *cache.RedisCache) *TrendingScorer {
	return &TrendingScorer{
		repo:  repo,
		cache: cache,
		done:  make(chan bool),
	}
}

// Start begins the background trending computation
func (ts *TrendingScorer) Start(ctx context.Context, interval time.Duration) {
	ts.ticker = time.NewTicker(interval)
	
	go func() {
		for {
			select {
			case <-ts.ticker.C:
				if err := ts.computeAllTiles(ctx); err != nil {
					log.Error().Err(err).Msg("Failed to compute trending tiles")
				}
			case <-ts.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	
	log.Info().Dur("interval", interval).Msg("Trending scorer started")
}

// Stop stops the background trending computation
func (ts *TrendingScorer) Stop() {
	if ts.ticker != nil {
		ts.ticker.Stop()
	}
	close(ts.done)
	log.Info().Msg("Trending scorer stopped")
}

// computeAllTiles computes trending scores for all active geohash tiles
func (ts *TrendingScorer) computeAllTiles(ctx context.Context) error {
	start := time.Now()
	
	// Get recent events (last 24 hours)
	since := time.Now().Add(-24 * time.Hour)
	events, err := ts.repo.GetRecentEventsByGeohash(ctx, since)
	if err != nil {
		return fmt.Errorf("failed to get recent events: %w", err)
	}
	
	if len(events) == 0 {
		log.Info().Msg("No recent events to compute trending scores")
		return nil
	}
	
	// Group events by geohash tiles
	tileEvents := ts.groupEventsByTile(events)
	
	// Compute scores for each tile
	tileCount := 0
	for geohash, tileEventList := range tileEvents {
		if err := ts.computeTileScore(ctx, geohash, tileEventList); err != nil {
			log.Warn().Err(err).Str("geohash", geohash).Msg("Failed to compute tile score")
			continue
		}
		tileCount++
	}
	
	// Update global trending metadata
	meta := TrendingMeta{
		LastComputedAt: time.Now(),
		EventCount:     len(events),
		TileCount:      tileCount,
	}
	
	globalMetaKey := "news:trending:global:meta"
	if data, err := json.Marshal(meta); err == nil {
		ts.cache.Set(ctx, globalMetaKey, data, cache.TrendingTTL)
	}
	
	log.Info().
		Dur("duration", time.Since(start)).
		Int("events", len(events)).
		Int("tiles", tileCount).
		Msg("Completed trending computation")
	
	return nil
}

// groupEventsByTile groups events by their geohash tile
func (ts *TrendingScorer) groupEventsByTile(events []repo.GetRecentEventsByGeohashRow) map[string][]repo.GetRecentEventsByGeohashRow {
	tileEvents := make(map[string][]repo.GetRecentEventsByGeohashRow)
	
	for _, event := range events {
		if event.UserLat == nil || event.UserLon == nil {
			continue
		}
		
		// Generate geohash for user location (precision 5)
		geohash := cache.GenerateGeohash(*event.UserLat, *event.UserLon, 5)
		tileEvents[geohash] = append(tileEvents[geohash], event)
	}
	
	return tileEvents
}

// computeTileScore computes trending score for a specific geohash tile
func (ts *TrendingScorer) computeTileScore(ctx context.Context, geohash string, events []repo.GetRecentEventsByGeohashRow) error {
	if len(events) == 0 {
		return nil
	}

	// Calculate trending scores for articles in this tile
	articleScores := make(map[string]float64)
	
	for _, event := range events {
		score := ts.calculateEventScore(event)
		articleScores[event.ArticleID] += score
	}

	// Convert to sorted list
	var trendingScores []TrendingScore
	for articleID, score := range articleScores {
		trendingScores = append(trendingScores, TrendingScore{
			ArticleID: articleID,
			Score:     score,
		})
	}

	// Sort by score (highest first)
	sort.Slice(trendingScores, func(i, j int) bool {
		return trendingScores[i].Score > trendingScores[j].Score
	})

	// Store in Redis ZSET
	trendingKey := cache.TrendingKey(geohash, 50) // Use default limit
	
	// Clear existing scores
	ts.cache.Del(ctx, trendingKey)
	
	// Add new scores
	for _, trendingScore := range trendingScores {
		ts.cache.ZAdd(ctx, trendingKey, redis.Z{
			Score:  trendingScore.Score,
			Member: trendingScore.ArticleID,
		})
	}
	
	// Set TTL
	ts.cache.Expire(ctx, trendingKey, cache.TrendingTTL)
	
	log.Info().
		Str("geohash", geohash).
		Int("events", len(events)).
		Int("articles", len(trendingScores)).
		Msg("Computed trending scores for tile")

	return nil
}

// calculateEventScore calculates the trending score for a single event
func (ts *TrendingScorer) calculateEventScore(event repo.GetRecentEventsByGeohashRow) float64 {
	// Event type weight
	var eventWeight float64
	switch event.Event {
	case "click":
		eventWeight = 2.0
	case "view":
		eventWeight = 1.0
	default:
		eventWeight = 1.0
	}
	
	// Time decay (exponential decay with 6-hour half-life)
	timeDiff := time.Since(event.OccurredAt)
	timeDecay := math.Exp(-timeDiff.Hours() / 6.0)
	
	// Geographic decay (if user location and article location available)
	var geoDecay float64 = 1.0
	if event.UserLat != nil && event.UserLon != nil && event.Latitude != nil && event.Longitude != nil {
		distance := ts.haversineDistance(*event.UserLat, *event.UserLon, *event.Latitude, *event.Longitude)
		geoDecay = 1.0 / (1.0 + distance/10.0) // 10km characteristic distance
	}
	
	// Final score
	score := eventWeight * timeDecay * geoDecay
	
	return score
}

// haversineDistance calculates the distance between two points using the Haversine formula
func (ts *TrendingScorer) haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
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

// SimulateUserEvents generates synthetic user events for testing and demonstration
func (ts *TrendingScorer) SimulateUserEvents(ctx context.Context) error {
	// Get some articles to create events for
	articles, err := ts.repo.GetArticlesByScore(ctx, repo.GetArticlesByScoreParams{
		Min:   0.5,
		Limit: 20,
	})
	if err != nil {
		return fmt.Errorf("failed to get articles for event simulation: %w", err)
	}
	
	if len(articles) == 0 {
		log.Info().Msg("No articles available for event simulation")
		return nil
	}
	
	// Generate random events
	eventCount := 0
	for i := 0; i < 50; i++ { // Generate 50 random events
		// Pick random article
		article := articles[rand.Intn(len(articles))]
		
		// Generate random user location near article location
		var userLat, userLon float64
		if article.Latitude != nil && article.Longitude != nil {
			// Random location within ~5km of article
			latOffset := (rand.Float64() - 0.5) * 0.05 // ~5km
			lonOffset := (rand.Float64() - 0.5) * 0.05
			userLat = *article.Latitude + latOffset
			userLon = *article.Longitude + lonOffset
		} else {
			// Random global location if article has no coordinates
			userLat = rand.Float64()*180 - 90
			userLon = rand.Float64()*360 - 180
		}
		
		// Random event type
		eventType := "view"
		if rand.Float64() < 0.3 { // 30% chance of click
			eventType = "click"
		}
		
		// Create event
		_, err := ts.repo.CreateUserEvent(ctx, repo.CreateUserEventParams{
			ArticleID: article.ID,
			Event:     eventType,
			UserLat:   &userLat,
			UserLon:   &userLon,
		})
		
		if err != nil {
			log.Warn().Err(err).Str("article_id", article.ID).Msg("Failed to create simulated event")
			continue
		}
		
		eventCount++
	}
	
	log.Info().Int("events_created", eventCount).Msg("Simulated user events")
	return nil
}

// GetTrendingScores retrieves trending scores for a geohash tile
func (ts *TrendingScorer) GetTrendingScores(ctx context.Context, geohash string, limit int) ([]TrendingScore, error) {
	trendingKey := cache.TrendingKey(geohash, limit)
	
	// Get top scores from Redis ZSET
	scores, err := ts.cache.ZRevRangeWithScores(ctx, trendingKey, 0, int64(limit-1))
	if err != nil {
		return nil, fmt.Errorf("failed to get trending scores: %w", err)
	}
	
	var trendingScores []TrendingScore
	for _, score := range scores {
		articleID, ok := score.Member.(string)
		if !ok {
			continue
		}
		trendingScores = append(trendingScores, TrendingScore{
			ArticleID: articleID,
			Score:     score.Score,
		})
	}
	
	return trendingScores, nil
}

// ForceRecompute forces recomputation of trending scores for a location
func (ts *TrendingScorer) ForceRecompute(ctx context.Context, lat, lon float64) error {
	geohash := cache.GenerateGeohash(lat, lon, 5)
	
	// Get recent events for this tile
	since := time.Now().Add(-24 * time.Hour) // Last 24 hours
	events, err := ts.repo.GetRecentEventsByGeohash(ctx, since)
	if err != nil {
		return fmt.Errorf("failed to get recent events: %w", err)
	}
	
	// Group events by tile
	tileEvents := ts.groupEventsByTile(events)
	
	// Compute score for this specific tile
	return ts.computeTileScore(ctx, geohash, tileEvents[geohash])
}
