package auth

import (
	"net/http"
	"strings"
)

// Middleware provides JWT authentication middleware.
type Middleware struct {
	jwtSecret string
}

// NewMiddleware creates a new auth middleware.
func NewMiddleware(jwtSecret string) *Middleware {
	return &Middleware{jwtSecret: jwtSecret}
}

// Verify is a chi-compatible middleware that checks for a valid Bearer token.
// Supports both Authorization header and ?token= query parameter (for SSE/EventSource).
func (m *Middleware) Verify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		// Try Authorization header first
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenStr = parts[1]
			}
		}

		// Fallback to query parameter (needed for EventSource/SSE)
		if tokenStr == "" {
			tokenStr = r.URL.Query().Get("token")
		}

		if tokenStr == "" {
			renderJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing authorization"})
			return
		}

		if err := ValidateToken(tokenStr, m.jwtSecret); err != nil {
			renderJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid or expired token"})
			return
		}

		next.ServeHTTP(w, r)
	})
}
