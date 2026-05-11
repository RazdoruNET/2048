package storage

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Models for database
type RequestLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	RequestID    string    `gorm:"uniqueIndex" json:"request_id"`
	Domain       string    `gorm:"index" json:"domain"`
	Method       string    `json:"method"`
	StatusCode   int       `json:"status_code"`
	Success      bool      `json:"success"`
	Latency      int64     `json:"latency"` // milliseconds
	BytesIn      int64     `json:"bytes_in"`
	BytesOut     int64     `json:"bytes_out"`
	Techniques   string    `json:"techniques"` // JSON array
	DPIType      string    `json:"dpi_type"`
	Confidence   float64   `json:"confidence"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type MLStatistics struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Timestamp         time.Time `gorm:"index" json:"timestamp"`
	TotalRequests     int       `json:"total_requests"`
	SuccessRate       float64   `json:"success_rate"`
	AverageLatency    int64     `json:"average_latency"`
	ThroughputBPS     int64     `json:"throughput_bps"`
	ErrorRate         float64   `json:"error_rate"`
	ActiveConnections  int       `json:"active_connections"`
}

type TechniqueEffectiveness struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Technique     string    `gorm:"index" json:"technique"`
	Domain        string    `gorm:"index" json:"domain"`
	Effectiveness float64   `json:"effectiveness"`
	SampleCount   int       `json:"sample_count"`
	LastUpdate    time.Time `json:"last_update"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type DPIEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RequestID    string    `gorm:"index" json:"request_id"`
	DPIType     string    `json:"dpi_type"`
	Confidence  float64   `json:"confidence"`
	Techniques   string    `json:"techniques"` // JSON array
	DetectedAt   time.Time `json:"detected_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type Storage struct {
	db *gorm.DB
}

func NewStorage() (*Storage, error) {
	db, err := gorm.Open(sqlite.Open("data/ml_monitoring.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&RequestLog{}, &MLStatistics{}, &TechniqueEffectiveness{}, &DPIEvent{})
	if err != nil {
		return nil, err
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Request logging
func (s *Storage) LogRequest(log *RequestLog) error {
	return s.db.Create(log).Error
}

func (s *Storage) GetRequests(limit int) ([]RequestLog, error) {
	var requests []RequestLog
	err := s.db.Order("created_at DESC").Limit(limit).Find(&requests).Error
	return requests, err
}

func (s *Storage) GetRequestByID(requestID string) (*RequestLog, error) {
	var request RequestLog
	err := s.db.Where("request_id = ?", requestID).First(&request).Error
	return &request, err
}

func (s *Storage) GetRequestsByDomain(domain string, limit int) ([]RequestLog, error) {
	var requests []RequestLog
	err := s.db.Where("domain = ?", domain).Order("created_at DESC").Limit(limit).Find(&requests).Error
	return requests, err
}

// ML Statistics
func (s *Storage) SaveMLStatistics(stats *MLStatistics) error {
	return s.db.Create(stats).Error
}

func (s *Storage) GetMLStatistics(hours int) ([]MLStatistics, error) {
	var stats []MLStatistics
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	err := s.db.Where("timestamp >= ?", since).Order("timestamp DESC").Find(&stats).Error
	return stats, err
}

func (s *Storage) GetLatestMLStatistics() (*MLStatistics, error) {
	var stats MLStatistics
	err := s.db.Order("timestamp DESC").First(&stats).Error
	return &stats, err
}

// Technique Effectiveness
func (s *Storage) UpdateTechniqueEffectiveness(tech *TechniqueEffectiveness) error {
	var existing TechniqueEffectiveness
	err := s.db.Where("technique = ? AND domain = ?", tech.Technique, tech.Domain).First(&existing).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create new record
		return s.db.Create(tech).Error
	} else if err != nil {
		return err
	}
	
	// Update existing record
	existing.Effectiveness = tech.Effectiveness
	existing.SampleCount = tech.SampleCount
	existing.LastUpdate = tech.LastUpdate
	return s.db.Save(&existing).Error
}

func (s *Storage) GetTechniqueEffectiveness(domain string) ([]TechniqueEffectiveness, error) {
	var techniques []TechniqueEffectiveness
	query := s.db
	if domain != "" {
		query = query.Where("domain = ?", domain)
	}
	err := query.Order("effectiveness DESC").Find(&techniques).Error
	return techniques, err
}

// DPI Events
func (s *Storage) LogDPIEvent(event *DPIEvent) error {
	return s.db.Create(event).Error
}

func (s *Storage) GetDPIEvents(requestID string) ([]DPIEvent, error) {
	var events []DPIEvent
	err := s.db.Where("request_id = ?", requestID).Order("detected_at ASC").Find(&events).Error
	return events, err
}

// Cleanup old data (older than 24 hours)
func (s *Storage) CleanupOldData() error {
	cutoff := time.Now().Add(-24 * time.Hour)

	// Clean old request logs
	s.db.Where("created_at < ?", cutoff).Delete(&RequestLog{})
	
	// Clean old ML statistics
	s.db.Where("timestamp < ?", cutoff).Delete(&MLStatistics{})
	
	// Clean old DPI events
	s.db.Where("created_at < ?", cutoff).Delete(&DPIEvent{})
	
	// Keep technique effectiveness data as it's valuable for learning
	
	return nil
}

// Domain statistics
func (s *Storage) GetDomainStatistics(domain string, hours int) (*DomainStats, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	
	var stats struct {
		TotalRequests    int64   `json:"total_requests"`
		SuccessfulReqs   int64   `json:"successful_requests"`
		TotalLatency     int64   `json:"total_latency"`
		TotalBytesIn     int64   `json:"total_bytes_in"`
		TotalBytesOut    int64   `json:"total_bytes_out"`
		AverageLatency   float64 `json:"average_latency"`
		SuccessRate      float64 `json:"success_rate"`
	}
	
	err := s.db.Model(&RequestLog{}).
		Select("COUNT(*) as total_requests, SUM(CASE WHEN success = true THEN 1 ELSE 0 END) as successful_requests, SUM(latency) as total_latency, SUM(bytes_in) as total_bytes_in, SUM(bytes_out) as total_bytes_out").
		Where("domain = ? AND created_at >= ?", domain, since).
		Scan(&stats).Error
	
	if err != nil {
		return nil, err
	}
	
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.SuccessfulReqs) / float64(stats.TotalRequests)
		stats.AverageLatency = float64(stats.TotalLatency) / float64(stats.TotalRequests)
	}
	
	return &DomainStats{
		Domain:          domain,
		TotalRequests:    int(stats.TotalRequests),
		SuccessRate:      stats.SuccessRate,
		AverageLatency:   stats.AverageLatency,
		TotalBytesIn:    stats.TotalBytesIn,
		TotalBytesOut:   stats.TotalBytesOut,
		TimeWindow:      hours,
	}, nil
}

type DomainStats struct {
	Domain         string  `json:"domain"`
	TotalRequests  int     `json:"total_requests"`
	SuccessRate    float64 `json:"success_rate"`
	AverageLatency float64 `json:"average_latency"`
	TotalBytesIn   int64   `json:"total_bytes_in"`
	TotalBytesOut  int64   `json:"total_bytes_out"`
	TimeWindow     int     `json:"time_window"`
}
