package models

import (
	"database/sql"
	"time"
)

type File struct {
	ID           int64     `json:"id"`
	JobID        string    `json:"job_id"`
	Filename    string    `json:"filename"`
	FileSize    int64     `json:"file_size"`
	FileType   string    `json:"file_type"`
	MimeType    string    `json:"mime_type"`
	CreatedAt   time.Time `json:"created_at"`
	ProcessedAt sql.NullTime `json:"processed_at"`
	Status     string    `json:"status"`
}

type MetadataRecord struct {
	ID         int64     `json:"id"`
	FileID    int64     `json:"file_id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Extension string    `json:"extension"`
	MimeType  string    `json:"mime_type"`
	CreatedAt time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	AccessedAt time.Time `json:"accessed_at"`
	Owner     string    `json:"owner"`
	Group     string    `json:"group"`
	Permissions string  `json:"permissions"`
}

type AnalysisResult struct {
	ID                int64     `json:"id"`
	FileID           int64     `json:"file_id"`
	ResultType       string    `json:"result_type"`
	StatisticName    string    `json:"statistic_name"`
	StatisticValue  float64   `json:"statistic_value"`
	Unit            string    `json:"unit"`
	CreatedAt        time.Time `json:"created_at"`
}

type Anomaly struct {
	ID          int64     `json:"id"`
	FileID     int64     `json:"file_id"`
	AnomalyType string   `json:"anomaly_type"`
	Severity    string   `json:"severity"`
	Description string  `json:"description"`
	Field      string   `json:"field"`
	Value      string   `json:"value"`
	Threshold  float64  `json:"threshold"`
	DetectedAt time.Time `json:"detected_at"`
}

type Job struct {
	JobID       string    `json:"job_id"`
	Status     string    `json:"status"`
	FileID     int64     `json:"file_id"`
	ErrorMsg   string    `json:"error_message"`
	CreatedAt  time.Time `json:"created_at"`
	StartedAt  time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// API Request/Response types

type UploadResponse struct {
	JobID   string `json:"job_id"`
	Status string `json:"status"`
}

type JobStatusResponse struct {
	JobID     string    `json:"job_id"`
	Status   string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Error    string    `json:"error,omitempty"`
}

type FileTypeStat struct {
	FileType string  `json:"file_type"`
	Count    int64   `json:"count"`
	Percent float64 `json:"percent"`
}

type SizeDistribution struct {
	Bucket string  `json:"bucket"`
	Count  int64   `json:"count"`
}

type TimeSeriesPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}