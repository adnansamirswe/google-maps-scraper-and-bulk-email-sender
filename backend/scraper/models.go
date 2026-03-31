package scraper

import (
	"time"
)

// FieldSelection defines which data fields the user wants to extract.
type FieldSelection struct {
	Phone        bool `json:"phone"`
	Email        bool `json:"email"`
	Website      bool `json:"website"`
	ReviewCount  bool `json:"review_count"`
	ReviewRating bool `json:"review_rating"`
	Address      bool `json:"address"`
	Category     bool `json:"category"`
	OpenHours    bool `json:"open_hours"`
	PriceRange   bool `json:"price_range"`
	Status       bool `json:"status"`
	Coordinates      bool `json:"coordinates"`
	Description      bool `json:"description"`
	ReviewsPerRating bool `json:"reviews_per_rating"`
}

// ScrapeRequest is the payload for creating a new scrape job.
type ScrapeRequest struct {
	Name           string         `json:"name"`
	Keywords       []string       `json:"keywords"`
	Language       string         `json:"language"`
	MaxDepth       int            `json:"max_depth"`
	Fields         FieldSelection `json:"fields"`
	Proxies        []string       `json:"proxies,omitempty"`
	GeoCoordinates string         `json:"geo,omitempty"`
	Zoom           int            `json:"zoom"`
	ExtractEmail   bool           `json:"extract_email"`
	WebhookURL     string         `json:"webhook_url,omitempty"`
	Concurrency    int            `json:"concurrency,omitempty"`
	MaxResults     int            `json:"max_results,omitempty"`
}

// JobStatus represents the current state of a scrape job.
type JobStatus string

const (
	StatusPending  JobStatus = "pending"
	StatusRunning  JobStatus = "running"
	StatusDone     JobStatus = "done"
	StatusFailed   JobStatus = "failed"
)

// Job represents a scrape job with its config and results.
type Job struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Status       JobStatus      `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Fields       FieldSelection `json:"fields"`
	Keywords     []string       `json:"keywords"`
	Language     string         `json:"language"`
	MaxDepth     int            `json:"max_depth"`
	Zoom         int            `json:"zoom"`
	Geo          string         `json:"geo,omitempty"`
	Proxies      []string       `json:"proxies,omitempty"`
	ExtractEmail   bool           `json:"extract_email"`
	PlacesFound  int            `json:"places_found"`
	WebhookURL   string         `json:"webhook_url,omitempty"`
	ErrorMsg     string         `json:"error_msg,omitempty"`
	Concurrency  int            `json:"concurrency,omitempty"`
	MaxResults   int            `json:"max_results,omitempty"`
}

// Place represents a single scraped business entry.
type Place struct {
	ID           string            `json:"id"`
	JobID        string            `json:"job_id"`
	Title        string            `json:"title"`
	Category     string            `json:"category,omitempty"`
	Address      string            `json:"address,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	Website      string            `json:"website,omitempty"`
	Email        string            `json:"email,omitempty"`
	ReviewCount  int               `json:"review_count,omitempty"`
	ReviewRating     float64           `json:"review_rating,omitempty"`
	ReviewsPerRating map[int]int       `json:"reviews_per_rating,omitempty"`
	Latitude         float64           `json:"latitude,omitempty"`
	Longitude    float64           `json:"longitude,omitempty"`
	Status       string            `json:"status,omitempty"`
	OpenHours    map[string]string `json:"open_hours,omitempty"`
	PriceRange   string            `json:"price_range,omitempty"`
	Description  string            `json:"description,omitempty"`
	Link         string            `json:"link,omitempty"`
	ScrapedAt    time.Time         `json:"scraped_at"`
}

// PlaceEvent is sent via SSE when a new place is found during scraping.
type PlaceEvent struct {
	Type  string `json:"type"` // "place", "progress", "done", "error"
	Place *Place `json:"place,omitempty"`
	Total int    `json:"total,omitempty"`
	Msg   string `json:"msg,omitempty"`
}

// SMTPConfig holds credentials and settings for an SMTP server.
type SMTPConfig struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	FromEmail string    `json:"from_email"`
	CreatedAt time.Time `json:"created_at"`
}

// BulkEmailPayload represents the data needed to send a bulk email blast.
type BulkEmailPayload struct {
	SMTPID    string   `json:"smtp_id"`
	Subject   string   `json:"subject"`
	BodyHTML  string   `json:"body_html"`
	Emails    []string `json:"emails"`
}
