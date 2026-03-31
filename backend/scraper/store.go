package scraper

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store handles SQLite persistence for jobs and places.
type Store struct {
	db      *sql.DB
	dataDir string
}

// NewStore creates a new SQLite store in the given data directory.
func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "scraper.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	store := &Store{db: db, dataDir: dataDir}

	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return store, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		fields_json TEXT NOT NULL,
		keywords_json TEXT NOT NULL,
		language TEXT NOT NULL DEFAULT 'en',
		max_depth INTEGER NOT NULL DEFAULT 10,
		zoom INTEGER NOT NULL DEFAULT 15,
		geo TEXT,
		proxies_json TEXT,
		extract_email BOOLEAN NOT NULL DEFAULT 0,
		places_found INTEGER NOT NULL DEFAULT 0,
		webhook_url TEXT,
		error_msg TEXT,
		concurrency INTEGER NOT NULL DEFAULT 1,
		max_results INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS places (
		id TEXT PRIMARY KEY,
		job_id TEXT NOT NULL,
		title TEXT NOT NULL,
		category TEXT,
		address TEXT,
		phone TEXT,
		website TEXT,
		email TEXT,
		review_count INTEGER DEFAULT 0,
		review_rating REAL DEFAULT 0,
		reviews_per_rating_json TEXT,
		latitude REAL DEFAULT 0,
		longitude REAL DEFAULT 0,
		status TEXT,
		open_hours_json TEXT,
		price_range TEXT,
		description TEXT,
		link TEXT,
		scraped_at DATETIME NOT NULL,
		FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_places_job_id ON places(job_id);

	CREATE TABLE IF NOT EXISTS smtp_configs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		host TEXT NOT NULL,
		port INTEGER NOT NULL,
		username TEXT NOT NULL,
		password TEXT NOT NULL,
		from_email TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return err
	}

	// Migrations (ignore errors for duplicate columns)
	_, _ = s.db.Exec("ALTER TABLE jobs ADD COLUMN webhook_url TEXT")
	_, _ = s.db.Exec("ALTER TABLE jobs ADD COLUMN concurrency INTEGER NOT NULL DEFAULT 1")
	_, _ = s.db.Exec("ALTER TABLE jobs ADD COLUMN max_results INTEGER NOT NULL DEFAULT 0")
	_, _ = s.db.Exec("ALTER TABLE places ADD COLUMN reviews_per_rating_json TEXT")

	return nil
}

// CreateJob inserts a new job into the database.
func (s *Store) CreateJob(job *Job) error {
	fieldsJSON, _ := json.Marshal(job.Fields)
	keywordsJSON, _ := json.Marshal(job.Keywords)
	proxiesJSON, _ := json.Marshal(job.Proxies)

	_, err := s.db.Exec(`
		INSERT INTO jobs (id, name, status, created_at, updated_at, fields_json, keywords_json, language, max_depth, zoom, geo, proxies_json, extract_email, webhook_url, concurrency, max_results)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.Name, job.Status, job.CreatedAt, job.UpdatedAt,
		string(fieldsJSON), string(keywordsJSON), job.Language,
		job.MaxDepth, job.Zoom, job.Geo, string(proxiesJSON), job.ExtractEmail, job.WebhookURL,
		job.Concurrency, job.MaxResults,
	)

	return err
}

// GetJob retrieves a single job by ID.
func (s *Store) GetJob(id string) (*Job, error) {
	row := s.db.QueryRow(`SELECT id, name, status, created_at, updated_at, fields_json, keywords_json, language, max_depth, zoom, geo, proxies_json, extract_email, places_found, webhook_url, error_msg, concurrency, max_results FROM jobs WHERE id = ?`, id)

	return scanJob(row)
}

// ListJobs returns all jobs ordered by creation date descending.
func (s *Store) ListJobs() ([]*Job, error) {
	rows, err := s.db.Query(`SELECT id, name, status, created_at, updated_at, fields_json, keywords_json, language, max_depth, zoom, geo, proxies_json, extract_email, places_found, webhook_url, error_msg, concurrency, max_results FROM jobs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job

	for rows.Next() {
		job, err := scanJobRow(rows)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	if jobs == nil {
		jobs = []*Job{}
	}

	return jobs, rows.Err()
}

// UpdateJobStatus updates the status and places_found count of a job.
func (s *Store) UpdateJobStatus(id string, status JobStatus, placesFound int, errorMsg string) error {
	_, err := s.db.Exec(`UPDATE jobs SET status = ?, places_found = ?, error_msg = ?, updated_at = ? WHERE id = ?`,
		status, placesFound, errorMsg, time.Now(), id)

	return err
}

// DeleteJob removes a job and its associated places.
func (s *Store) DeleteJob(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM places WHERE job_id = ?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err := tx.Exec(`DELETE FROM jobs WHERE id = ?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// InsertPlace adds a scraped place to the database.
func (s *Store) InsertPlace(place *Place) error {
	hoursJSON, _ := json.Marshal(place.OpenHours)
	reviewsPerRatingJSON, _ := json.Marshal(place.ReviewsPerRating)

	_, err := s.db.Exec(`
		INSERT INTO places (id, job_id, title, category, address, phone, website, email, review_count, review_rating, reviews_per_rating_json, latitude, longitude, status, open_hours_json, price_range, description, link, scraped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		place.ID, place.JobID, place.Title, place.Category, place.Address,
		place.Phone, place.Website, place.Email, place.ReviewCount, place.ReviewRating,
		string(reviewsPerRatingJSON), place.Latitude, place.Longitude, place.Status, string(hoursJSON),
		place.PriceRange, place.Description, place.Link, place.ScrapedAt,
	)

	return err
}

// GetPlaces retrieves all places for a given job.
func (s *Store) GetPlaces(jobID string) ([]*Place, error) {
	rows, err := s.db.Query(`SELECT id, job_id, title, category, address, phone, website, email, review_count, review_rating, reviews_per_rating_json, latitude, longitude, status, open_hours_json, price_range, description, link, scraped_at FROM places WHERE job_id = ? ORDER BY scraped_at ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var places []*Place

	for rows.Next() {
		var p Place
		var hoursJSON, reviewsRatingJSON sql.NullString

		err := rows.Scan(&p.ID, &p.JobID, &p.Title, &p.Category, &p.Address,
			&p.Phone, &p.Website, &p.Email, &p.ReviewCount, &p.ReviewRating,
			&reviewsRatingJSON, &p.Latitude, &p.Longitude, &p.Status, &hoursJSON,
			&p.PriceRange, &p.Description, &p.Link, &p.ScrapedAt)
		if err != nil {
			return nil, err
		}

		if hoursJSON.Valid {
			_ = json.Unmarshal([]byte(hoursJSON.String), &p.OpenHours)
		}

		if reviewsRatingJSON.Valid {
			_ = json.Unmarshal([]byte(reviewsRatingJSON.String), &p.ReviewsPerRating)
		}

		places = append(places, &p)
	}

	if places == nil {
		places = []*Place{}
	}

	return places, rows.Err()
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// scanner interface for both *sql.Row and *sql.Rows
type scanner interface {
	Scan(dest ...any) error
}

func scanJobFromScanner(sc scanner) (*Job, error) {
	var job Job
	var fieldsJSON, keywordsJSON, proxiesJSON sql.NullString
	var geo, webhook, errorMsg sql.NullString
	var concurrency, maxResults sql.NullInt64

	err := sc.Scan(&job.ID, &job.Name, &job.Status, &job.CreatedAt, &job.UpdatedAt,
		&fieldsJSON, &keywordsJSON, &job.Language, &job.MaxDepth, &job.Zoom,
		&geo, &proxiesJSON, &job.ExtractEmail, &job.PlacesFound, &webhook, &errorMsg,
		&concurrency, &maxResults)
	if err != nil {
		return nil, err
	}

	if fieldsJSON.Valid {
		_ = json.Unmarshal([]byte(fieldsJSON.String), &job.Fields)
	}

	if keywordsJSON.Valid {
		_ = json.Unmarshal([]byte(keywordsJSON.String), &job.Keywords)
	}

	if proxiesJSON.Valid {
		_ = json.Unmarshal([]byte(proxiesJSON.String), &job.Proxies)
	}

	if geo.Valid {
		job.Geo = geo.String
	}

	if webhook.Valid {
		job.WebhookURL = webhook.String
	}

	if errorMsg.Valid {
		job.ErrorMsg = errorMsg.String
	}

	if concurrency.Valid {
		job.Concurrency = int(concurrency.Int64)
	} else {
		job.Concurrency = 1 // default
	}

	if maxResults.Valid {
		job.MaxResults = int(maxResults.Int64)
	}

	return &job, nil
}

func scanJob(row *sql.Row) (*Job, error) {
	return scanJobFromScanner(row)
}

func scanJobRow(rows *sql.Rows) (*Job, error) {
	return scanJobFromScanner(rows)
}
