package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {
	host := getEnv("DB_HOST", "postgres")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "scraper_db")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var db *sql.DB
	var err error
	maxRetries := 10
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", psqlInfo)
		if err != nil {
			log.Printf("Attempt %d/%d: Failed to open database connection: %v", i+1, maxRetries, err)
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
			}
			continue
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		err = db.Ping()
		if err != nil {
			log.Printf("Attempt %d/%d: Failed to ping database: %v", i+1, maxRetries, err)
			db.Close()
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
			}
			continue
		}

		log.Println("Successfully connected to database")
		return db, nil
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %v", maxRetries, err)
}

func InitSchema(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sources (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			url VARCHAR(500) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS data_entries (
			id SERIAL PRIMARY KEY,
			source_id INTEGER REFERENCES sources(id) ON DELETE CASCADE,
			title VARCHAR(500) NOT NULL,
			cleaned_content TEXT NOT NULL,
			share_date TIMESTAMP,
			criticality_score INTEGER DEFAULT 0 CHECK (criticality_score >= 0 AND criticality_score <= 100),
			category VARCHAR(100) DEFAULT 'Uncategorized',
			ai_analysis TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`DO $$ 
		BEGIN 
			IF EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name='data_entries' AND column_name='raw_content') THEN
				ALTER TABLE data_entries DROP COLUMN raw_content;
			END IF;
		END $$;`,
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_data_entries_source_id ON data_entries(source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_data_entries_category ON data_entries(category)`,
		`CREATE INDEX IF NOT EXISTS idx_data_entries_criticality ON data_entries(criticality_score)`,
		`CREATE INDEX IF NOT EXISTS idx_data_entries_created_at ON data_entries(created_at)`,
		`DO $$ 
		BEGIN 
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name='data_entries' AND column_name='ai_analysis') THEN
				ALTER TABLE data_entries ADD COLUMN ai_analysis TEXT;
			END IF;
		END $$;`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %v", err)
		}
	}

	defaultPassword := getEnv("ADMIN_PASSWORD", "admin123")
	hashedPassword, _ := hashPassword(defaultPassword)
	_, err := db.Exec(`
		INSERT INTO users (username, password_hash) 
		VALUES ('admin', $1) 
		ON CONFLICT (username) DO NOTHING
	`, hashedPassword)

	if err != nil {
		log.Printf("Warning: Could not create default admin user: %v", err)
	}

	log.Println("Database schema initialized successfully")
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

