package telemetry

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/alexandrmotologa/framesieve/internal/stream"
	_ "modernc.org/sqlite"
)

// CumulativeStats aggregates global telemetry totals across sessions.
type CumulativeStats struct {
	TotalSessions    int64   `json:"total_sessions"`
	TotalFrames      int64   `json:"total_frames"`
	PassedFrames     int64   `json:"passed_frames"`
	DroppedFrames    int64   `json:"dropped_frames"`
	DropRatio        float64 `json:"drop_ratio"`
	TotalTokensNaive int64   `json:"total_tokens_naive"`
	TotalTokensSent  int64   `json:"total_tokens_sent"`
	TokensSaved      int64   `json:"tokens_saved"`
	SavedRatio       float64 `json:"saved_ratio"`
	EstDollarsSaved  float64 `json:"est_dollars_saved"`
	AvgLatencyUs     float64 `json:"avg_latency_us"`
}

// StoredMetric represents an indexed frame event in SQLite.
type StoredMetric struct {
	ID          int64     `json:"id"`
	SessionID   string    `json:"session_id"`
	Timestamp   time.Time `json:"timestamp"`
	FrameID     int64     `json:"frame_id"`
	SSIM        float64   `json:"ssim"`
	HasROI      bool      `json:"has_roi"`
	RoiX        int       `json:"roi_x"`
	RoiY        int       `json:"roi_y"`
	RoiW        int       `json:"roi_w"`
	RoiH        int       `json:"roi_h"`
	TokensSent  int       `json:"tokens_sent"`
	TokensSaved int       `json:"tokens_saved"`
	LatencyUs   int64     `json:"latency_us"`
	Dropped     bool      `json:"dropped"`
	DropCode    string    `json:"drop_code"`
}

// TimeSeriesPoint represents a point on the live activity timeline.
type TimeSeriesPoint struct {
	Timestamp   string  `json:"timestamp"`
	SSIM        float64 `json:"ssim"`
	TokensSaved int64   `json:"tokens_saved"`
	TokensSent  int64   `json:"tokens_sent"`
	LatencyUs   int64   `json:"latency_us"`
}

// Store wraps the SQLite database handle.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// Open initializes SQLite tables and indices.
func Open(dbPath string) (*Store, error) {
	if dbPath == "" {
		dbPath = "framesieve.db"
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) initSchema() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	CREATE TABLE IF NOT EXISTS sessions (
		session_id TEXT PRIMARY KEY,
		started_at TIMESTAMP NOT NULL,
		ended_at TIMESTAMP,
		source_type TEXT NOT NULL,
		total_frames INTEGER DEFAULT 0,
		dropped_frames INTEGER DEFAULT 0,
		tokens_saved INTEGER DEFAULT 0,
		dollars_saved REAL DEFAULT 0.0
	);

	CREATE TABLE IF NOT EXISTS frame_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		frame_id INTEGER NOT NULL,
		ssim REAL NOT NULL,
		has_roi INTEGER NOT NULL,
		roi_x INTEGER DEFAULT 0,
		roi_y INTEGER DEFAULT 0,
		roi_w INTEGER DEFAULT 0,
		roi_h INTEGER DEFAULT 0,
		tokens_sent INTEGER DEFAULT 0,
		tokens_saved INTEGER DEFAULT 0,
		latency_us INTEGER DEFAULT 0,
		dropped INTEGER NOT NULL,
		drop_code TEXT DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_metrics_session ON frame_metrics(session_id);
	CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON frame_metrics(timestamp);
	`
	_, err := s.db.Exec(query)
	return err
}

// Close gracefully closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// CreateSession logs the start of a streaming or proxy session.
func (s *Store) CreateSession(sessionID, sourceType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `INSERT INTO sessions (session_id, started_at, source_type) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, sessionID, time.Now(), sourceType)
	return err
}

// RecordMetric writes a single frame telemetry point.
func (s *Store) RecordMetric(sessionID string, evt stream.FrameEvent, latency time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var hasROI int
	var rx, ry, rw, rh int
	if evt.ROI != nil {
		hasROI = 1
		rx = evt.ROI.X
		ry = evt.ROI.Y
		rw = evt.ROI.Width
		rh = evt.ROI.Height
	}

	var dropped int
	var dropCode string
	if evt.TriggerFrame != nil {
		if evt.TriggerFrame.Dropped {
			dropped = 1
			dropCode = evt.TriggerFrame.DropCode
		}
	}

	query := `
	INSERT INTO frame_metrics (
		session_id, timestamp, frame_id, ssim, has_roi,
		roi_x, roi_y, roi_w, roi_h,
		tokens_sent, tokens_saved, latency_us, dropped, drop_code
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	frameID := int64(0)
	if evt.TriggerFrame != nil {
		frameID = evt.TriggerFrame.ID
	}

	_, err := s.db.Exec(
		query,
		sessionID, evt.Timestamp, frameID, evt.SSIM, hasROI,
		rx, ry, rw, rh,
		evt.TokensSent, evt.TokensSaved, latency.Microseconds(), dropped, dropCode,
	)
	return err
}

// GetCumulativeStats computes lifetime performance numbers.
func (s *Store) GetCumulativeStats() (CumulativeStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	SELECT 
		COUNT(DISTINCT session_id),
		COUNT(*),
		SUM(CASE WHEN dropped = 0 THEN 1 ELSE 0 END),
		SUM(CASE WHEN dropped = 1 THEN 1 ELSE 0 END),
		COALESCE(SUM(tokens_sent), 0),
		COALESCE(SUM(tokens_saved), 0),
		COALESCE(AVG(latency_us), 0)
	FROM frame_metrics`

	var stats CumulativeStats
	row := s.db.QueryRow(query)
	err := row.Scan(
		&stats.TotalSessions,
		&stats.TotalFrames,
		&stats.PassedFrames,
		&stats.DroppedFrames,
		&stats.TotalTokensSent,
		&stats.TokensSaved,
		&stats.AvgLatencyUs,
	)
	if err != nil {
		return stats, err
	}

	stats.TotalTokensNaive = stats.TotalTokensSent + stats.TokensSaved
	if stats.TotalFrames > 0 {
		stats.DropRatio = float64(stats.DroppedFrames) / float64(stats.TotalFrames)
	}
	if stats.TotalTokensNaive > 0 {
		stats.SavedRatio = float64(stats.TokensSaved) / float64(stats.TotalTokensNaive)
	}
	stats.EstDollarsSaved = float64(stats.TokensSaved) * 0.0000025

	return stats, nil
}

// GetRecentMetrics returns the latest N metrics ordered descending.
func (s *Store) GetRecentMetrics(limit int) ([]StoredMetric, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		limit = 50
	}

	query := `
	SELECT id, session_id, timestamp, frame_id, ssim, has_roi,
	       roi_x, roi_y, roi_w, roi_h, tokens_sent, tokens_saved,
	       latency_us, dropped, drop_code
	FROM frame_metrics
	ORDER BY id DESC
	LIMIT ?`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []StoredMetric
	for rows.Next() {
		var m StoredMetric
		var hasROI, dropped int
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.Timestamp, &m.FrameID, &m.SSIM, &hasROI,
			&m.RoiX, &m.RoiY, &m.RoiW, &m.RoiH, &m.TokensSent, &m.TokensSaved,
			&m.LatencyUs, &dropped, &m.DropCode,
		); err != nil {
			return nil, err
		}
		m.HasROI = hasROI == 1
		m.Dropped = dropped == 1
		res = append(res, m)
	}

	return res, nil
}
