package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"

	"github.com/go-chi/chi/v5"
)

// ListSMTPConfigsHandler returns all stored SMTP profiles.
func (h *Handler) ListSMTPConfigsHandler(w http.ResponseWriter, r *http.Request) {
	configs, err := h.store.ListSMTPConfigs()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list SMTP configs"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configs)
}

// CreateSMTPConfigHandler securely binds and stores a new SMTP backend.
func (h *Handler) CreateSMTPConfigHandler(w http.ResponseWriter, r *http.Request) {
	var cfg SMTPConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}

	if cfg.Name == "" || cfg.Host == "" || cfg.Username == "" || cfg.FromEmail == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required fields"})
		return
	}

	if err := h.store.CreateSMTPConfig(&cfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create config"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cfg)
}

// DeleteSMTPConfigHandler securely removes an SMTP profile by ID.
func (h *Handler) DeleteSMTPConfigHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id param"})
		return
	}

	if err := h.store.DeleteSMTPConfig(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete config"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateSMTPConfigHandler handles PUT requests to modify an existing SMTP configuration.
func (h *Handler) UpdateSMTPConfigHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id param"})
		return
	}

	var cfg SMTPConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	cfg.ID = id

	if cfg.Name == "" || cfg.Host == "" || cfg.Username == "" || cfg.FromEmail == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required fields"})
		return
	}

	if err := h.store.UpdateSMTPConfig(&cfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update config"})
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// SendBulkEmailHandler triggers a massive asynchronous SMTP mailing cycle natively looping through all targets.
func (h *Handler) SendBulkEmailHandler(w http.ResponseWriter, r *http.Request) {
	var payload BulkEmailPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}

	if payload.SMTPID == "" || payload.Subject == "" || payload.BodyHTML == "" || len(payload.Emails) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required fields (smtp_id, subject, body_html, emails)"})
		return
	}

	smtpConfig, err := h.store.GetSMTPConfig(payload.SMTPID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to identify mapped smtp configuration"})
		return
	}

	// Trigger async routine to avoid locking down the browser UI response cycle
	go func(cfg *SMTPConfig, data BulkEmailPayload) {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		hostAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

		successCount := 0
		failCount := 0

		for _, recipient := range data.Emails {
			msg := "To: " + recipient + "\r\n" +
				"From: " + cfg.FromEmail + "\r\n" +
				"Subject: " + data.Subject + "\r\n" +
				"MIME-version: 1.0;\r\n" +
				"Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n" +
				data.BodyHTML

			err := smtp.SendMail(hostAddr, auth, cfg.FromEmail, []string{recipient}, []byte(msg))
			if err != nil {
				log.Printf("[Emailer] Failed sending to %s via %s: %v", recipient, cfg.Host, err)
				failCount++
			} else {
				successCount++
			}
		}

		log.Printf("[Emailer] Bulk Blast Complete: %d delivered, %d failed.", successCount, failCount)
	}(smtpConfig, payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "accepted",
		"msg":    fmt.Sprintf("Mailing operation explicitly loaded. Processing %d targets asynchronously.", len(payload.Emails)),
	})
}
