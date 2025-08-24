package middleware

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog/log"
)

// Recovery middleware to handle panics
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				log.Error().
					Interface("panic", err).
					Str("stack", string(debug.Stack())).
					Str("url", r.URL.String()).
					Str("method", r.Method).
					Str("remote_addr", r.RemoteAddr).
					Msg("Panic recovered")
				
				// Return 500 error
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				
				errorResponse := map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "INTERNAL_ERROR",
						"message": "Internal server error",
					},
				}
				
				if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}

