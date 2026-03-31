package scraper

import (
	"time"

	"github.com/google/uuid"
)

// CreateSMTPConfig securely saves a new SMTP sender profile.
func (s *Store) CreateSMTPConfig(cfg *SMTPConfig) error {
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = time.Now()
	}

	_, err := s.db.Exec(`
		INSERT INTO smtp_configs (id, name, host, port, username, password, from_email, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		cfg.ID, cfg.Name, cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.FromEmail, cfg.CreatedAt,
	)
	return err
}

// ListSMTPConfigs loads all connected SMTP accounts ordered by creation.
func (s *Store) ListSMTPConfigs() ([]*SMTPConfig, error) {
	rows, err := s.db.Query(`SELECT id, name, host, port, username, password, from_email, created_at FROM smtp_configs ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*SMTPConfig
	for rows.Next() {
		cfg := &SMTPConfig{}
		if err := rows.Scan(&cfg.ID, &cfg.Name, &cfg.Host, &cfg.Port, &cfg.Username, &cfg.Password, &cfg.FromEmail, &cfg.CreatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}

	if configs == nil {
		configs = []*SMTPConfig{}
	}

	return configs, nil
}

// GetSMTPConfig grabs an active SMTP configuration mapping by its ID.
func (s *Store) GetSMTPConfig(id string) (*SMTPConfig, error) {
	row := s.db.QueryRow(`SELECT id, name, host, port, username, password, from_email, created_at FROM smtp_configs WHERE id = ?`, id)
	
	cfg := &SMTPConfig{}
	err := row.Scan(&cfg.ID, &cfg.Name, &cfg.Host, &cfg.Port, &cfg.Username, &cfg.Password, &cfg.FromEmail, &cfg.CreatedAt)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// DeleteSMTPConfig removes an integrated SMTP sender dynamically.
func (s *Store) DeleteSMTPConfig(id string) error {
	_, err := s.db.Exec(`DELETE FROM smtp_configs WHERE id = ?`, id)
	return err
}

// UpdateSMTPConfig modifies an existing SMTP sender profile.
func (s *Store) UpdateSMTPConfig(cfg *SMTPConfig) error {
	_, err := s.db.Exec(`
		UPDATE smtp_configs 
		SET name = ?, host = ?, port = ?, username = ?, password = ?, from_email = ?
		WHERE id = ?`,
		cfg.Name, cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.FromEmail, cfg.ID,
	)
	return err
}
