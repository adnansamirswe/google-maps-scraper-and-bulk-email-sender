package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler manages scraper API endpoints.
type Handler struct {
	store   *Store
	engine  *Engine
	mu      sync.RWMutex
	streams map[string][]chan PlaceEvent // jobID -> list of SSE channels
}

// NewHandler creates a new scraper handler.
func NewHandler(store *Store) *Handler {
	return &Handler{
		store:   store,
		engine:  NewEngine(),
		streams: make(map[string][]chan PlaceEvent),
	}
}

// Routes registers all scraper routes on the given chi router.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/api/scrape", h.Create)
	r.Get("/api/jobs", h.List)
	r.Get("/api/jobs/{id}", h.Get)
	r.Delete("/api/jobs/{id}", h.Delete)
	r.Get("/api/jobs/{id}/places", h.GetPlaces)
	r.Get("/api/jobs/{id}/stream", h.Stream)

	// SMTP & Emailer
	r.Get("/api/smtp", h.ListSMTPConfigsHandler)
	r.Post("/api/smtp", h.CreateSMTPConfigHandler)
	r.Put("/api/smtp/{id}", h.UpdateSMTPConfigHandler)
	r.Delete("/api/smtp/{id}", h.DeleteSMTPConfigHandler)
	r.Post("/api/emailer/send", h.SendBulkEmailHandler)
}

// Create handles POST /api/scrape — creates a new scrape job.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req ScrapeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.Keywords) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "keywords are required"})
		return
	}

	if req.Language == "" {
		req.Language = "en"
	}

	if req.MaxDepth == 0 {
		req.MaxDepth = 10
	}

	if req.Concurrency <= 0 {
		req.Concurrency = 1
	}

	if req.Zoom == 0 {
		req.Zoom = 15
	}

	if req.Name == "" {
		req.Name = req.Keywords[0]
	}

	now := time.Now()
	job := &Job{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Status:       StatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
		Fields:       req.Fields,
		Keywords:     req.Keywords,
		Language:     req.Language,
		MaxDepth:     req.MaxDepth,
		Zoom:         req.Zoom,
		Geo:          req.GeoCoordinates,
		Proxies:      req.Proxies,
		ExtractEmail: req.ExtractEmail,
		WebhookURL:   req.WebhookURL,
		Concurrency:  req.Concurrency,
		MaxResults:   req.MaxResults,
	}

	if err := h.store.CreateJob(job); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create job"})
		return
	}

	// Start the real scraper engine in the background
	go h.engine.RunScrape(context.Background(), h, job)

	writeJSON(w, http.StatusCreated, job)
}

// List handles GET /api/jobs — lists all jobs.
func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	jobs, err := h.store.ListJobs()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list jobs"})
		return
	}

	writeJSON(w, http.StatusOK, jobs)
}

// Get handles GET /api/jobs/{id} — returns a single job.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	job, err := h.store.GetJob(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}

	writeJSON(w, http.StatusOK, job)
}

// Delete handles DELETE /api/jobs/{id} — deletes a job and its places.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.store.DeleteJob(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete job"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetPlaces handles GET /api/jobs/{id}/places — returns places for a job.
func (h *Handler) GetPlaces(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	places, err := h.store.GetPlaces(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get places"})
		return
	}

	writeJSON(w, http.StatusOK, places)
}

// Stream handles GET /api/jobs/{id}/stream — SSE endpoint for real-time updates.
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan PlaceEvent, 50)

	h.mu.Lock()
	h.streams[id] = append(h.streams[id], ch)
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		channels := h.streams[id]

		for i, c := range channels {
			if c == ch {
				h.streams[id] = append(channels[:i], channels[i+1:]...)

				break
			}
		}

		h.mu.Unlock()
		close(ch)
	}()

	// Send initial connection event
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"msg\":\"stream connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}

			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			if event.Type == "done" || event.Type == "error" {
				return
			}
		}
	}
}

// broadcast sends an event to all SSE subscribers of a job.
func (h *Handler) broadcast(jobID string, event PlaceEvent) {
	h.mu.RLock()
	channels := h.streams[jobID]
	h.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
			// Skip if channel buffer is full
		}
	}
}

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(data)
}
