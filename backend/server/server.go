package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"gmaps-scraper/auth"
	"gmaps-scraper/config"
	"gmaps-scraper/scraper"
)

// Server is the main HTTP server.
type Server struct {
	srv    *http.Server
	cfg    *config.Config
	store  *scraper.Store
}

// New creates and configures the HTTP server.
func New(cfg *config.Config) (*Server, error) {
	store, err := scraper.NewStore(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Auth handler (public)
	authHandler := auth.NewHandler(cfg.Password, cfg.JWTSecret)
	r.Post("/api/auth/login", authHandler.Login)

	// Health check (public)
	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Protected routes
	authMiddleware := auth.NewMiddleware(cfg.JWTSecret)
	scraperHandler := scraper.NewHandler(store)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Verify)
		scraperHandler.Routes(r)
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      0, // Disable for SSE
		IdleTimeout:       120 * time.Second,
	}

	return &Server{
		srv:   srv,
		cfg:   cfg,
		store: store,
	}, nil
}

// Start runs the HTTP server and blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	}()

	log.Printf("🚀 Server starting on http://localhost:%s", s.cfg.Port)

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Close cleans up server resources.
func (s *Server) Close() error {
	return s.store.Close()
}
