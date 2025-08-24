package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

// Logging middleware for zerolog
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Get request ID from context
		requestID := middleware.GetReqID(r.Context())
		
		// Create logger with request context
		logger := log.With().
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Logger()
		
		// Log request start
		logger.Info().Msg("Request started")
		
		// Wrap response writer to capture status code
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		
		// Process request
		next.ServeHTTP(ww, r)
		
		// Calculate duration
		duration := time.Since(start)
		
		// Log request completion
		logger.Info().
			Int("status", ww.Status()).
			Int("bytes", ww.BytesWritten()).
			Dur("duration", duration).
			Msg("Request completed")
		
		// Log errors for 4xx and 5xx status codes
		if ww.Status() >= 400 {
			logger.Error().
				Int("status", ww.Status()).
				Dur("duration", duration).
				Msg("Request failed")
		}
	})
}
