package service

import (
	"database/sql"
	"fmt"
	"time"
)

type SourceService struct {
	db *sql.DB
}

func NewSourceService(db *sql.DB) *SourceService {
	return &SourceService{db: db}
}

type Source struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *SourceService) GetAllSources() ([]Source, error) {
	rows, err := s.db.Query(`
		SELECT id, name, url, created_at 
		FROM sources 
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		var source Source
		if err := rows.Scan(&source.ID, &source.Name, &source.URL, &source.CreatedAt); err != nil {
			continue
		}
		sources = append(sources, source)
	}

	return sources, nil
}

func (s *SourceService) GetSourceByID(id int) (*Source, error) {
	var source Source
	err := s.db.QueryRow(`
		SELECT id, name, url, created_at 
		FROM sources 
		WHERE id = $1
	`, id).Scan(&source.ID, &source.Name, &source.URL, &source.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &source, nil
}

func (s *SourceService) CreateSource(name, url string) (*Source, error) {
	if name == "" {
		return nil, fmt.Errorf("source name is required")
	}
	if url == "" {
		return nil, fmt.Errorf("source URL is required")
	}

	var source Source
	err := s.db.QueryRow(`
		INSERT INTO sources (name, url) 
		VALUES ($1, $2) 
		RETURNING id, name, url, created_at
	`, name, url).Scan(&source.ID, &source.Name, &source.URL, &source.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &source, nil
}

func (s *SourceService) UpdateSource(id int, name, url string) error {
	if name == "" {
		return fmt.Errorf("source name is required")
	}
	if url == "" {
		return fmt.Errorf("source URL is required")
	}

	_, err := s.db.Exec(`
		UPDATE sources 
		SET name = $1, url = $2 
		WHERE id = $3
	`, name, url, id)

	return err
}

func (s *SourceService) DeleteSource(id int) error {
	_, err := s.db.Exec(`DELETE FROM sources WHERE id = $1`, id)
	return err
}

