package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/RITWIZSINGH/DoodleDash-backend/internal/config"
	"github.com/RITWIZSINGH/DoodleDash-backend/internal/models"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"golang.org/x/time/rate"
)

// ApplyMiddleware applies all middleware to the router
func ApplyMiddleware(router *mux.Router, cfg *config.Config) http.Handler {
	// CORS middleware
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		AllowCredentials: true,
	})

	// Rate limiting middleware
	limiter := rate.NewLimiter(rate.Limit(cfg.RateLimit.RequestsPerMinute/60), cfg.RateLimit.BurstSize)

	return corsMiddleware.Handler(rateLimitMiddleware(limiter, loggingMiddleware(router)))
}

// loggingMiddleware logs incoming requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// rateLimitMiddleware applies rate limiting
func rateLimitMiddleware(limiter *rate.Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := limiter.Wait(r.Context()); err != nil {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware checks for valid authentication (simplified for guest users)
func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In a real application, check for JWT or session token
			// For this example, we'll allow guest access
			next.ServeHTTP(w, r)
		})
	}
}

// GenerateGuestUser creates a new guest user
func GenerateGuestUser() *models.User {
	return models.NewGuestUser()
}