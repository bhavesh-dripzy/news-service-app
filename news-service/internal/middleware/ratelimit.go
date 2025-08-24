package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
	}
}

// RateLimit middleware for basic rate limiting
// Note: This is a simplified implementation. In production, you'd want to use Redis
// for distributed rate limiting across multiple instances.
func RateLimit(next http.Handler) http.Handler {
	config := DefaultRateLimitConfig()
	
	// Simple in-memory rate limiter (not suitable for production with multiple instances)
	// In production, use Redis-based rate limiting
	limiter := NewSimpleRateLimiter(config.RequestsPerMinute, config.BurstSize)
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		clientIP := getClientIP(r)
		
		// Check rate limit
		if !limiter.Allow(clientIP) {
			log.Warn().
				Str("client_ip", clientIP).
				Str("url", r.URL.String()).
				Msg("Rate limit exceeded")
			
			// Return rate limit error
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			
			errorResponse := map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "RATE_LIMIT",
					"message": "Rate limit exceeded. Please try again later.",
				},
			}
			
			if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			}
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the real client IP address
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// Take the first IP in the chain
		if commaIndex := indexOf(forwardedFor, ','); commaIndex > 0 {
			return forwardedFor[:commaIndex]
		}
		return forwardedFor
	}
	
	// Check X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	
	// Fall back to remote address
	return r.RemoteAddr
}

// indexOf finds the index of a character in a string
func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// SimpleRateLimiter is a basic in-memory rate limiter
// Not suitable for production with multiple instances
type SimpleRateLimiter struct {
	requestsPerMinute int
	burstSize         int
	clients           map[string]*clientLimit
}

type clientLimit struct {
	tokens     int
	lastRefill time.Time
}

func NewSimpleRateLimiter(requestsPerMinute, burstSize int) *SimpleRateLimiter {
	return &SimpleRateLimiter{
		requestsPerMinute: requestsPerMinute,
		burstSize:         burstSize,
		clients:           make(map[string]*clientLimit),
	}
}

func (rl *SimpleRateLimiter) Allow(clientIP string) bool {
	now := time.Now()
	
	// Get or create client limit
	client, exists := rl.clients[clientIP]
	if !exists {
		client = &clientLimit{
			tokens:     rl.burstSize,
			lastRefill: now,
		}
		rl.clients[clientIP] = client
	}
	
	// Refill tokens based on time passed
	timePassed := now.Sub(client.lastRefill)
	tokensToAdd := int(timePassed.Minutes() * float64(rl.requestsPerMinute))
	
	if tokensToAdd > 0 {
		client.tokens = min(client.tokens+tokensToAdd, rl.burstSize)
		client.lastRefill = now
	}
	
	// Check if we have tokens
	if client.tokens > 0 {
		client.tokens--
		return true
	}
	
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
