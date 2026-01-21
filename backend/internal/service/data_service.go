package service

import (
	"database/sql"
	"fmt"
	"time"
)

type DataService struct {
	db *sql.DB
}

func NewDataService(db *sql.DB) *DataService {
	return &DataService{db: db}
}

func (s *DataService) GetDB() *sql.DB {
	return s.db
}

type DataEntry struct {
	ID              int        `json:"id"`
	SourceID        int        `json:"source_id"`
	SourceName      string     `json:"source_name"`
	SourceURL       string     `json:"source_url"`
	Title           string     `json:"title"`
	CleanedContent  string     `json:"cleaned_content"`
	ShareDate       *time.Time `json:"share_date"`
	CriticalityScore int       `json:"criticality_score"`
	Category        string     `json:"category"`
	AIAnalysis      *string    `json:"ai_analysis,omitempty"` // Optional AI interpretation
	CreatedAt       time.Time  `json:"created_at"`
}

type CategoryStats struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type CriticalityDistribution struct {
	Range  string `json:"range"`
	Count  int    `json:"count"`
}

type DashboardStats struct {
	TotalEntries      int                      `json:"total_entries"`
	TotalSources      int                      `json:"total_sources"`
	CategoryStats     []CategoryStats          `json:"category_stats"`
	CriticalityDist   []CriticalityDistribution `json:"criticality_distribution"`
	RecentEntries     []DataEntry              `json:"recent_entries"`
	TimeSeriesData    []TimeSeriesData         `json:"time_series_data,omitempty"`
	AIAnalysisStatus  *AIAnalysisStatus        `json:"ai_analysis_status,omitempty"`
}

func (s *DataService) GetAllEntries(page, pageSize int, category, search string) ([]DataEntry, int, error) {
	offset := (page - 1) * pageSize

	query := `
		SELECT e.id, e.source_id, s.name, s.url, e.title, e.cleaned_content, 
		       e.share_date, e.criticality_score, e.category, e.ai_analysis, e.created_at
		FROM data_entries e
		JOIN sources s ON e.source_id = s.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if category != "" {
		query += fmt.Sprintf(" AND e.category = $%d", argIndex)
		args = append(args, category)
		argIndex++
	}

	if search != "" {
		query += fmt.Sprintf(" AND (e.title ILIKE $%d OR e.cleaned_content ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	query += " ORDER BY e.created_at DESC"
	
	countQuery := "SELECT COUNT(*) FROM (" + query + ") as count_query"
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []DataEntry
	for rows.Next() {
		var entry DataEntry
		var shareDate sql.NullTime
		var aiAnalysis sql.NullString
		err := rows.Scan(
			&entry.ID, &entry.SourceID, &entry.SourceName, &entry.SourceURL,
			&entry.Title, &entry.CleanedContent, &shareDate,
			&entry.CriticalityScore, &entry.Category, &aiAnalysis, &entry.CreatedAt,
		)
		if err != nil {
			continue
		}
		if shareDate.Valid {
			entry.ShareDate = &shareDate.Time
		}
		if aiAnalysis.Valid && aiAnalysis.String != "" {
			entry.AIAnalysis = &aiAnalysis.String
		}
		entries = append(entries, entry)
	}

	return entries, total, nil
}

func (s *DataService) GetEntryByID(id int) (*DataEntry, error) {
	var entry DataEntry
	var shareDate sql.NullTime
	var aiAnalysis sql.NullString

	err := s.db.QueryRow(`
		SELECT e.id, e.source_id, s.name, s.url, e.title, e.cleaned_content,
		       e.share_date, e.criticality_score, e.category, e.ai_analysis, e.created_at
		FROM data_entries e
		JOIN sources s ON e.source_id = s.id
		WHERE e.id = $1
	`, id).Scan(
		&entry.ID, &entry.SourceID, &entry.SourceName, &entry.SourceURL,
		&entry.Title, &entry.CleanedContent, &shareDate,
		&entry.CriticalityScore, &entry.Category, &aiAnalysis, &entry.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	if shareDate.Valid {
		entry.ShareDate = &shareDate.Time
	}

	if aiAnalysis.Valid && aiAnalysis.String != "" {
		entry.AIAnalysis = &aiAnalysis.String
	}

	return &entry, nil
}

func (s *DataService) UpdateCriticality(id int, score int) error {
	if score < 0 || score > 100 {
		return fmt.Errorf("criticality score must be between 0 and 100")
	}

	_, err := s.db.Exec(`
		UPDATE data_entries 
		SET criticality_score = $1 
		WHERE id = $2
	`, score, id)

	return err
}

func (s *DataService) UpdateCategory(id int, category string) error {
	_, err := s.db.Exec(`
		UPDATE data_entries 
		SET category = $1 
		WHERE id = $2
	`, category, id)

	return err
}

func (s *DataService) GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Total entries
	err := s.db.QueryRow("SELECT COUNT(*) FROM data_entries").Scan(&stats.TotalEntries)
	if err != nil {
		return nil, err
	}

	// Total sources
	err = s.db.QueryRow("SELECT COUNT(*) FROM sources").Scan(&stats.TotalSources)
	if err != nil {
		return nil, err
	}

	// Category stats
	rows, err := s.db.Query(`
		SELECT category, COUNT(*) as count
		FROM data_entries
		GROUP BY category
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cs CategoryStats
		if err := rows.Scan(&cs.Category, &cs.Count); err == nil {
			stats.CategoryStats = append(stats.CategoryStats, cs)
		}
	}

	// Criticality distribution
	criticalityRanges := []struct {
		label string
		min   int
		max   int
	}{
		{"0-20", 0, 20},
		{"21-40", 21, 40},
		{"41-60", 41, 60},
		{"61-80", 61, 80},
		{"81-100", 81, 100},
	}

	for _, r := range criticalityRanges {
		var count int
		s.db.QueryRow(`
			SELECT COUNT(*) 
			FROM data_entries 
			WHERE criticality_score >= $1 AND criticality_score <= $2
		`, r.min, r.max).Scan(&count)

		stats.CriticalityDist = append(stats.CriticalityDist, CriticalityDistribution{
			Range: r.label,
			Count: count,
		})
	}

	// Recent entries (last 10)
	rows, err = s.db.Query(`
		SELECT e.id, e.source_id, s.name, s.url, e.title, e.cleaned_content,
		       e.share_date, e.criticality_score, e.category, e.ai_analysis, e.created_at
		FROM data_entries e
		JOIN sources s ON e.source_id = s.id
		ORDER BY e.created_at DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry DataEntry
		var shareDate sql.NullTime
		var aiAnalysis sql.NullString
		if err := rows.Scan(
			&entry.ID, &entry.SourceID, &entry.SourceName, &entry.SourceURL,
			&entry.Title, &entry.CleanedContent, &shareDate,
			&entry.CriticalityScore, &entry.Category, &aiAnalysis, &entry.CreatedAt,
		); err == nil {
			if shareDate.Valid {
				entry.ShareDate = &shareDate.Time
			}
			if aiAnalysis.Valid && aiAnalysis.String != "" {
				entry.AIAnalysis = &aiAnalysis.String
			}
			stats.RecentEntries = append(stats.RecentEntries, entry)
		}
	}

	// Time series data (last 30 days)
	timeSeriesData, err := s.GetTimeSeriesData(30)
	if err == nil {
		stats.TimeSeriesData = timeSeriesData
	}

	// AI analysis status
	aiStatus, err := s.GetAIAnalysisStatus()
	if err == nil {
		stats.AIAnalysisStatus = aiStatus
	}

	return stats, nil
}

func (s *DataService) GetCategories() ([]string, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT category 
		FROM data_entries 
		ORDER BY category
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err == nil {
			categories = append(categories, cat)
		}
	}

	return categories, nil
}

// TimeSeriesData represents time-based entry counts
type TimeSeriesData struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// AIAnalysisStatus represents AI analysis completion status
type AIAnalysisStatus struct {
	WithAnalysis    int `json:"with_analysis"`
	WithoutAnalysis int `json:"without_analysis"`
}

// GetTimeSeriesData returns entry counts grouped by share_date (for trend analysis)
func (s *DataService) GetTimeSeriesData(days int) ([]TimeSeriesData, error) {
	query := `
		SELECT 
			DATE(share_date) as date,
			COUNT(*) as count
		FROM data_entries
		WHERE share_date IS NOT NULL
		  AND share_date >= CURRENT_DATE - INTERVAL '1 day' * $1
		GROUP BY DATE(share_date)
		ORDER BY date ASC
	`
	
	rows, err := s.db.Query(query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var data []TimeSeriesData
	for rows.Next() {
		var ts TimeSeriesData
		var date sql.NullTime
		if err := rows.Scan(&date, &ts.Count); err == nil {
			if date.Valid {
				ts.Date = date.Time.Format("2006-01-02")
				data = append(data, ts)
			}
		}
	}
	
	return data, nil
}

// GetAIAnalysisStatus returns count of entries with and without AI analysis
func (s *DataService) GetAIAnalysisStatus() (*AIAnalysisStatus, error) {
	status := &AIAnalysisStatus{}
	
	// Count entries with AI analysis
	err := s.db.QueryRow(`
		SELECT COUNT(*) 
		FROM data_entries 
		WHERE ai_analysis IS NOT NULL AND ai_analysis != ''
	`).Scan(&status.WithAnalysis)
	if err != nil {
		return nil, err
	}
	
	// Count entries without AI analysis
	err = s.db.QueryRow(`
		SELECT COUNT(*) 
		FROM data_entries 
		WHERE ai_analysis IS NULL OR ai_analysis = ''
	`).Scan(&status.WithoutAnalysis)
	if err != nil {
		return nil, err
	}
	
	return status, nil
}

