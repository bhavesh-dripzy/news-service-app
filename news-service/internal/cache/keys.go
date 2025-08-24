package cache

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"time"
)

const (
	ArticleTTL        = 6 * time.Hour
	SummaryTTL        = 7 * 24 * time.Hour
	SearchTTL         = 90 * time.Second
	CategoryTTL       = 2 * time.Hour
	SourceTTL         = 2 * time.Hour
	ScoreTTL          = 2 * time.Hour
	NearbyTTL         = 5 * time.Minute
	TrendingTTL       = 2 * time.Minute
	GeohashTTL        = 1 * time.Hour
	UserEventTTL      = 24 * time.Hour
)

// ArticleKey generates Redis key for article cache
func ArticleKey(id string) string {
	return fmt.Sprintf("news:article:%s", id)
}

// SummaryKey generates Redis key for article summary cache
func SummaryKey(id string) string {
	return fmt.Sprintf("news:summary:%s", id)
}

// SearchKey generates Redis key for search results cache
func SearchKey(query string, limit int) string {
	hash := sha1.Sum([]byte(fmt.Sprintf("%s|%d", query, limit)))
	return fmt.Sprintf("cache:v1:search:%x", hash)
}

// CategoryKey generates Redis key for category results cache
func CategoryKey(name string, limit int) string {
	hash := sha1.Sum([]byte(fmt.Sprintf("category:%s:%d", name, limit)))
	return fmt.Sprintf("cache:v1:category:%x", hash)
}

// SourceKey generates Redis key for source results cache
func SourceKey(name string, limit int) string {
	hash := sha1.Sum([]byte(fmt.Sprintf("source:%s:%d", name, limit)))
	return fmt.Sprintf("cache:v1:source:%x", hash)
}

// ScoreKey generates Redis key for score results cache
func ScoreKey(min float64, limit int) string {
	hash := sha1.Sum([]byte(fmt.Sprintf("score:%.2f:%d", min, limit)))
	return fmt.Sprintf("cache:v1:score:%x", hash)
}

// NearbyKey generates Redis key for nearby results cache
func NearbyKey(lat, lon, radius float64, limit int) string {
	hash := sha1.Sum([]byte(fmt.Sprintf("nearby:%.6f:%.6f:%.1f:%d", lat, lon, radius, limit)))
	return fmt.Sprintf("cache:v1:nearby:%x", hash)
}

// TrendingKey generates Redis key for trending results cache
func TrendingKey(geohash string, limit int) string {
	return fmt.Sprintf("trending:geohash:%s:limit:%d", geohash, limit)
}

// GeohashKey generates Redis key for geohash data
func GeohashKey(geohash string) string {
	return fmt.Sprintf("geo:hash:%s", geohash)
}

// UserEventKey generates Redis key for user events
func UserEventKey(articleID string) string {
	return fmt.Sprintf("events:article:%s", articleID)
}

// RateLimitKey generates Redis key for rate limiting
func RateLimitKey(clientIP string) string {
	return fmt.Sprintf("ratelimit:ip:%s", clientIP)
}

// Helper function to generate geohash from lat/lon
// This is a simplified implementation - in production, use a proper geohash library
func GenerateGeohash(lat, lon float64, precision int) string {
	// Simplified geohash implementation
	// In production, use github.com/mmcloughlin/geohash or similar
	
	// Base32 characters for geohash
	const base32 = "0123456789bcdefghjkmnpqrstuvwxyz"
	
	// Simple hash-based approach for demo purposes
	// This is NOT a proper geohash implementation
	latHash := int((lat + 90.0) * 1000000) % 1000000
	lonHash := int((lon + 180.0) * 1000000) % 1000000
	
	combined := latHash*1000000 + lonHash
	geohash := ""
	
	for i := 0; i < precision; i++ {
		geohash += string(base32[combined%32])
		combined /= 32
	}
	
	return geohash
}

// ParseGeohash parses a geohash back to lat/lon (simplified)
func ParseGeohash(geohash string) (float64, float64, error) {
	// This is a simplified implementation
	// In production, use a proper geohash library
	
	if len(geohash) == 0 {
		return 0, 0, fmt.Errorf("empty geohash")
	}
	
	// Simple reverse hash for demo purposes
	// This is NOT accurate geohash parsing
	hash := 0
	for i, char := range geohash {
		hash += int(char) * (i + 1)
	}
	
	// Convert hash back to approximate coordinates
	lat := float64(hash%180000-90000) / 1000.0
	lon := float64(hash%360000-180000) / 1000.0
	
	return lat, lon, nil
}

// GeohashBoundingBox returns the bounding box for a geohash
func GeohashBoundingBox(geohash string) (float64, float64, float64, float64, error) {
	// This is a simplified implementation
	// In production, use a proper geohash library
	
	lat, lon, err := ParseGeohash(geohash)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	
	// Approximate bounding box (very rough)
	precision := len(geohash)
	// Use simple division instead of bit shift to avoid type issues
	latDelta := 180.0 / float64(precision*precision)
	lonDelta := 360.0 / float64(precision*precision)
	
	minLat := lat - latDelta/2
	maxLat := lat + latDelta/2
	minLon := lon - lonDelta/2
	maxLon := lon + lonDelta/2
	
	return minLat, minLon, maxLat, maxLon, nil
}

// GetTTL returns the appropriate TTL for a given key
func GetTTL(key string) time.Duration {
	switch {
	case strings.Contains(key, "news:article:"):
		return ArticleTTL
	case strings.Contains(key, "news:summary:"):
		return SummaryTTL
	case strings.Contains(key, "cache:v1:search:"):
		return SearchTTL
	case strings.Contains(key, "cache:v1:category:"):
		return CategoryTTL
	case strings.Contains(key, "cache:v1:source:"):
		return SourceTTL
	case strings.Contains(key, "cache:v1:score:"):
		return ScoreTTL
	case strings.Contains(key, "cache:v1:nearby:"):
		return NearbyTTL
	case strings.Contains(key, "trending:geohash:"):
		return TrendingTTL
	case strings.Contains(key, "geo:hash:"):
		return GeohashTTL
	case strings.Contains(key, "events:article:"):
		return UserEventTTL
	default:
		return 5 * time.Minute // default TTL
	}
}
