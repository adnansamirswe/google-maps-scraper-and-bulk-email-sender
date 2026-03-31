package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/scrapemate"
	"github.com/gosom/scrapemate/scrapemateapp"
)

// Engine wraps the real Google Maps scraper.
type Engine struct{}

// NewEngine creates a new scraper engine.
func NewEngine() *Engine {
	return &Engine{}
}

// sseWriter implements scrapemate.ResultWriter to intercept results
// and broadcast them in real-time via SSE.
type sseWriter struct {
	handler *Handler
	jobID   string
	fields  FieldSelection
	count   int
	cancel  context.CancelFunc
	maxR    int
}

func (w *sseWriter) Run(_ context.Context, in <-chan scrapemate.Result) error {
	for result := range in {
		entry, ok := result.Data.(*gmaps.Entry)
		if !ok {
			continue
		}

		w.count++

		place := entryToPlace(entry, w.jobID, w.fields)

		_ = w.handler.store.InsertPlace(place)
		_ = w.handler.store.UpdateJobStatus(w.jobID, StatusRunning, w.count, "")

		w.handler.broadcast(w.jobID, PlaceEvent{
			Type:  "place",
			Place: place,
			Total: w.count,
		})

		if w.maxR > 0 && w.count >= w.maxR {
			if w.cancel != nil {
				w.cancel()
			}
		}
	}

	return nil
}

// RunScrape runs the actual Google Maps scraper for a job.
func (e *Engine) RunScrape(ctx context.Context, handler *Handler, job *Job) {
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("Scraper engine crashed unexpectedly: %v", r)
			_ = handler.store.UpdateJobStatus(job.ID, StatusFailed, 0, errStr)
			dispatchWebhook(job, StatusFailed, 0, errStr)
			handler.broadcast(job.ID, PlaceEvent{
				Type: "error",
				Msg:  errStr,
			})
		}
	}()
	_ = handler.store.UpdateJobStatus(job.ID, StatusRunning, 0, "")

	handler.broadcast(job.ID, PlaceEvent{
		Type: "progress",
		Msg:  "Initializing scraper engine...",
	})

	scrapeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	writer := &sseWriter{
		handler: handler,
		jobID:   job.ID,
		fields:  job.Fields,
		cancel:  cancel,
		maxR:    job.MaxResults,
	}

	// Configure scrapemate options
	opts := []func(*scrapemateapp.Config) error{
		scrapemateapp.WithConcurrency(job.Concurrency),
		scrapemateapp.WithJS(scrapemateapp.DisableImages()),
	}

	if len(job.Proxies) > 0 {
		opts = append(opts, scrapemateapp.WithProxies(job.Proxies))
	}

	matecfg, err := scrapemateapp.NewConfig(
		[]scrapemate.ResultWriter{writer},
		opts...,
	)
	if err != nil {
		handler.broadcast(job.ID, PlaceEvent{
			Type: "error",
			Msg:  fmt.Sprintf("Failed to configure scraper: %v", err),
		})

		_ = handler.store.UpdateJobStatus(job.ID, StatusFailed, 0, err.Error())
		dispatchWebhook(job, StatusFailed, 0, err.Error())

		return
	}

	app, err := scrapemateapp.NewScrapeMateApp(matecfg)
	if err != nil {
		handler.broadcast(job.ID, PlaceEvent{
			Type: "error",
			Msg:  fmt.Sprintf("Failed to create scraper: %v", err),
		})

		_ = handler.store.UpdateJobStatus(job.ID, StatusFailed, 0, err.Error())
		dispatchWebhook(job, StatusFailed, 0, err.Error())

		return
	}

	defer app.Close()

	// Create seed jobs from keywords
	var seedJobs []scrapemate.IJob

	for _, keyword := range job.Keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}

		gmapJob := gmaps.NewGmapJob(
			uuid.New().String(),
			job.Language,
			keyword,
			job.MaxDepth,
			job.ExtractEmail,
			job.Geo,
			job.Zoom,
		)

		seedJobs = append(seedJobs, gmapJob)
	}

	if len(seedJobs) == 0 {
		handler.broadcast(job.ID, PlaceEvent{
			Type: "error",
			Msg:  "No valid keywords provided",
		})

		_ = handler.store.UpdateJobStatus(job.ID, StatusFailed, 0, "no valid keywords")
		dispatchWebhook(job, StatusFailed, 0, "no valid keywords")

		return
	}

	handler.broadcast(job.ID, PlaceEvent{
		Type: "progress",
		Msg:  fmt.Sprintf("Scraping %d keywords with Playwright browser...", len(seedJobs)),
	})

	// Run the scraper

	if err := app.Start(scrapeCtx, seedJobs...); err != nil {
		if scrapeCtx.Err() != nil {
			// Context was cancelled — not an error
			_ = handler.store.UpdateJobStatus(job.ID, StatusDone, writer.count, "")
		} else {
			handler.broadcast(job.ID, PlaceEvent{
				Type: "error",
				Msg:  fmt.Sprintf("Scraper error: %v", err),
			})

			_ = handler.store.UpdateJobStatus(job.ID, StatusFailed, writer.count, err.Error())
			dispatchWebhook(job, StatusFailed, writer.count, err.Error())

			return
		}
	}

	_ = handler.store.UpdateJobStatus(job.ID, StatusDone, writer.count, "")
	dispatchWebhook(job, StatusDone, writer.count, "")

	handler.broadcast(job.ID, PlaceEvent{
		Type:  "done",
		Total: writer.count,
		Msg:   fmt.Sprintf("Scrape completed — %d places found", writer.count),
	})
}

// entryToPlace converts a gmaps.Entry to our Place model, filtering by selected fields.
func entryToPlace(entry *gmaps.Entry, jobID string, fields FieldSelection) *Place {
	place := &Place{
		ID:        uuid.New().String(),
		JobID:     jobID,
		Title:     entry.Title,
		Link:      entry.Link,
		ScrapedAt: time.Now(),
	}

	if fields.Category {
		place.Category = entry.Category
	}

	if fields.Address {
		place.Address = entry.Address
	}

	if fields.Phone {
		place.Phone = entry.Phone
	}

	if fields.Website {
		place.Website = entry.WebSite
	}

	if fields.Email {
		if len(entry.Emails) > 0 {
			var valid []string
			for _, e := range entry.Emails {
				lower := strings.ToLower(e)
				if !strings.HasSuffix(lower, ".png") &&
					!strings.HasSuffix(lower, ".jpg") &&
					!strings.HasSuffix(lower, ".jpeg") &&
					!strings.HasSuffix(lower, ".webp") &&
					!strings.HasSuffix(lower, ".gif") &&
					!strings.HasSuffix(lower, ".svg") &&
					!strings.HasSuffix(lower, ".bmp") &&
					!strings.HasSuffix(lower, ".tiff") &&
					!strings.HasPrefix(lower, "logo") &&
					!strings.Contains(lower, "@2x") &&
					!strings.Contains(lower, "@3x") {
					valid = append(valid, e)
				}
			}
			if len(valid) > 0 {
				place.Email = strings.Join(valid, ", ")
			}
		}
	}

	if fields.ReviewCount {
		place.ReviewCount = entry.ReviewCount
	}

	if fields.ReviewRating {
		place.ReviewRating = entry.ReviewRating
	}

	if fields.ReviewsPerRating {
		place.ReviewsPerRating = entry.ReviewsPerRating
	}

	if fields.Coordinates {
		place.Latitude = entry.Latitude
		place.Longitude = entry.Longtitude
	}

	if fields.Status {
		place.Status = entry.Status
	}

	if fields.OpenHours {
		if entry.OpenHours != nil {
			place.OpenHours = make(map[string]string)
			for day, hours := range entry.OpenHours {
				place.OpenHours[day] = strings.Join(hours, ", ")
			}
		}
	}

	if fields.PriceRange {
		place.PriceRange = entry.PriceRange
	}

	if fields.Description {
		place.Description = entry.Description
	}

	return place
}

// dispatchWebhook sends an HTTP POST to the webhook URL upon job completion.
func dispatchWebhook(job *Job, finalStatus JobStatus, count int, errorMsg string) {
	if job.WebhookURL == "" {
		return
	}

	payload := map[string]interface{}{
		"id":           job.ID,
		"name":         job.Name,
		"status":       finalStatus,
		"places_found": count,
	}
	if errorMsg != "" {
		payload["error_msg"] = errorMsg
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", job.WebhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
		}
	}()
}
