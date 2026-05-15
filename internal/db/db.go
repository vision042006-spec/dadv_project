package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn                 *sql.DB
	PasswordResetTokens   map[string]struct {
		Token   string
		Expires time.Time
	}
}

func New(dsn string) (*DB, error) {
	// Ensure directory exists
	dir := dsn
	if i := len(dir) - 1; i > 0 {
		for ; dir[i] != '/' && i > 0; i-- {
		}
		if i > 0 {
			dir = dir[:i]
			os.MkdirAll(dir, 0755)
		}
	}

	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	db := &DB{
		conn: conn,
		PasswordResetTokens: make(map[string]struct {
			Token   string
			Expires time.Time
		}),
	}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	if err := db.InitUsersTable(context.Background()); err != nil {
		return nil, fmt.Errorf("users table migration failed: %w", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL DEFAULT 0,
		job_id TEXT UNIQUE NOT NULL,
		filename TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		file_type TEXT,
		mime_type TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		processed_at DATETIME,
		status TEXT DEFAULT 'pending'
	);

	CREATE TABLE IF NOT EXISTS metadata_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		path TEXT,
		size INTEGER NOT NULL,
		extension TEXT,
		mime_type TEXT,
		created_at DATETIME,
		modified_at DATETIME,
		accessed_at DATETIME,
		owner TEXT,
		group_name TEXT,
		permissions TEXT,
		FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS analysis_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id INTEGER NOT NULL,
		result_type TEXT NOT NULL,
		statistic_name TEXT NOT NULL,
		statistic_value REAL NOT NULL,
		unit TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS anomalies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id INTEGER NOT NULL,
		anomaly_type TEXT NOT NULL,
		severity TEXT NOT NULL,
		description TEXT NOT NULL,
		field TEXT,
		value TEXT,
		threshold REAL,
		detected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_files_user_id ON files(user_id);
	CREATE INDEX IF NOT EXISTS idx_files_job_id ON files(job_id);
	CREATE INDEX IF NOT EXISTS idx_files_status ON files(status);
	CREATE INDEX IF NOT EXISTS idx_metadata_file_id ON metadata_records(file_id);
	CREATE INDEX IF NOT EXISTS idx_metadata_extension ON metadata_records(extension);
	CREATE INDEX IF NOT EXISTS idx_metadata_size ON metadata_records(size);
	CREATE INDEX IF NOT EXISTS idx_results_file_id ON analysis_results(file_id);
	CREATE INDEX IF NOT EXISTS idx_anomalies_file_id ON anomalies(file_id);
	`
	_, err := db.conn.Exec(schema)
	return err
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}

// File operations
func (db *DB) CreateFile(ctx context.Context, userID int64, jobID, filename string, size int64, mimeType string) (int64, error) {
	query := `INSERT INTO files (user_id, job_id, filename, file_size, mime_type, status) VALUES (?, ?, ?, ?, ?, 'pending')`
	result, err := db.conn.ExecContext(ctx, query, userID, jobID, filename, size, mimeType)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) UpdateFileStatus(ctx context.Context, jobID, status string) error {
	query := `UPDATE files SET status = ?, processed_at = CURRENT_TIMESTAMP WHERE job_id = ?`
	_, err := db.conn.ExecContext(ctx, query, status, jobID)
	return err
}

func (db *DB) GetFileByJobID(ctx context.Context, userID int64, jobID string) (int64, string, error) {
	query := `SELECT id, status FROM files WHERE job_id = ? AND (user_id = ? OR user_id = 0)`
	var id int64
	var status string
	err := db.conn.QueryRowContext(ctx, query, jobID, userID).Scan(&id, &status)
	return id, status, err
}

func (db *DB) ListFiles(ctx context.Context, userID int64) ([]map[string]interface{}, error) {
	query := `SELECT id, job_id, filename, file_size, mime_type, status, created_at FROM files WHERE user_id = ? OR user_id = 0 ORDER BY created_at DESC LIMIT 50`
	rows, err := db.conn.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var files []map[string]interface{}
	for rows.Next() {
		var id int64
		var jobID, filename, mimeType, status, createdAt string
		var fileSize int64
		if err := rows.Scan(&id, &jobID, &filename, &fileSize, &mimeType, &status, &createdAt); err != nil {
			return nil, err
		}
		files = append(files, map[string]interface{}{
			"id": id,
			"job_id": jobID,
			"filename": filename,
			"file_size": fileSize,
			"mime_type": mimeType,
			"status": status,
			"created_at": createdAt,
		})
	}
	return files, nil
}

// Metadata operations
func (db *DB) InsertMetadataBatch(ctx context.Context, records []map[string]interface{}) error {
	if len(records) == 0 {
		return nil
	}
	
	query := `INSERT INTO metadata_records 
		(file_id, name, path, size, extension, mime_type, created_at, modified_at, accessed_at, owner, group_name, permissions)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	for _, r := range records {
		_, err := stmt.ExecContext(ctx,
			r["file_id"],
			r["name"],
			r["path"],
			r["size"],
			r["extension"],
			r["mime_type"],
			r["created_at"],
			r["modified_at"],
			r["accessed_at"],
			r["owner"],
			r["group"],
			r["permissions"],
		)
		if err != nil {
			return err
		}
	}
	
	return tx.Commit()
}

func (db *DB) GetMetadataCount(ctx context.Context, fileID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM metadata_records WHERE file_id = ?`
	var count int64
	err := db.conn.QueryRowContext(ctx, query, fileID).Scan(&count)
	return count, err
}

// Analysis operations
func (db *DB) InsertAnalysisResult(ctx context.Context, fileID int64, resultType, name string, value float64, unit string) error {
	query := `INSERT INTO analysis_results (file_id, result_type, statistic_name, statistic_value, unit) VALUES (?, ?, ?, ?, ?)`
	_, err := db.conn.ExecContext(ctx, query, fileID, resultType, name, value, unit)
	return err
}

func (db *DB) GetAnalysisResults(ctx context.Context, fileID int64) ([]map[string]interface{}, error) {
	query := `SELECT result_type, statistic_name, statistic_value, unit, created_at FROM analysis_results WHERE file_id = ? ORDER BY created_at`
	rows, err := db.conn.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []map[string]interface{}
	for rows.Next() {
		var resultType, statisticName, unit, createdAt string
		var statisticValue float64
		if err := rows.Scan(&resultType, &statisticName, &statisticValue, &unit, &createdAt); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"result_type": resultType,
			"statistic_name": statisticName,
			"statistic_value": statisticValue,
			"unit": unit,
			"created_at": createdAt,
		})
	}
	return results, nil
}

// Anomaly operations
func (db *DB) InsertAnomaly(ctx context.Context, fileID int64, anomalyType, severity, desc, field, value string, threshold float64) error {
	query := `INSERT INTO anomalies (file_id, anomaly_type, severity, description, field, value, threshold) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := db.conn.ExecContext(ctx, query, fileID, anomalyType, severity, desc, field, value, threshold)
	return err
}

func (db *DB) GetAnomalies(ctx context.Context, fileID int64) ([]map[string]interface{}, error) {
	query := `SELECT anomaly_type, severity, description, field, value, threshold, detected_at FROM anomalies WHERE file_id = ? ORDER BY detected_at DESC`
	rows, err := db.conn.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var anomalies []map[string]interface{}
	for rows.Next() {
		var anomalyType, severity, desc, field, value string
		var threshold, detectedAt interface{}
		if err := rows.Scan(&anomalyType, &severity, &desc, &field, &value, &threshold, &detectedAt); err != nil {
			return nil, err
		}
		anomalies = append(anomalies, map[string]interface{}{
			"anomaly_type": anomalyType,
			"severity": severity,
			"description": desc,
			"field": field,
			"value": value,
			"threshold": threshold,
			"detected_at": detectedAt,
		})
	}
	return anomalies, nil
}

// Stats queries
func (db *DB) GetFileTypeStats(ctx context.Context, fileID int64) ([]map[string]interface{}, error) {
	query := `
		SELECT extension as file_type, COUNT(*) as count, 
			CAST(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM metadata_records WHERE file_id = ?) AS FLOAT) as percent
		FROM metadata_records 
		WHERE file_id = ? AND extension IS NOT NULL
		GROUP BY extension
		ORDER BY count DESC
		LIMIT 20`
	rows, err := db.conn.QueryContext(ctx, query, fileID, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var stats []map[string]interface{}
	for rows.Next() {
		var fileType string
		var count int64
		var percent float64
		if err := rows.Scan(&fileType, &count, &percent); err != nil {
			return nil, err
		}
		stats = append(stats, map[string]interface{}{
			"file_type": fileType,
			"count": count,
			"percent": percent,
		})
	}
	return stats, nil
}

func (db *DB) GetSizeDistribution(ctx context.Context, fileID int64) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			CASE 
				WHEN size < 1024 THEN '< 1KB'
				WHEN size < 1024*1024 THEN '1KB - 1MB'
				WHEN size < 10*1024*1024 THEN '1MB - 10MB'
				WHEN size < 100*1024*1024 THEN '10MB - 100MB'
				ELSE '> 100MB'
			END as bucket,
			COUNT(*) as count
		FROM metadata_records 
		WHERE file_id = ?
		GROUP BY bucket
		ORDER BY bucket`
	rows, err := db.conn.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var dist []map[string]interface{}
	for rows.Next() {
		var bucket string
		var count int64
		if err := rows.Scan(&bucket, &count); err != nil {
			return nil, err
		}
		dist = append(dist, map[string]interface{}{
			"bucket": bucket,
			"count": count,
		})
	}
	return dist, nil
}

func (db *DB) GetOwnershipStats(ctx context.Context, fileID int64) ([]map[string]interface{}, error) {
	query := `
		SELECT owner, COUNT(*) as count, SUM(size) as total_size
		FROM metadata_records 
		WHERE file_id = ? AND owner IS NOT NULL
		GROUP BY owner
		ORDER BY count DESC
		LIMIT 10`
	rows, err := db.conn.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var stats []map[string]interface{}
	for rows.Next() {
		var owner string
		var count int64
		var totalSize int64
		if err := rows.Scan(&owner, &count, &totalSize); err != nil {
			return nil, err
		}
		stats = append(stats, map[string]interface{}{
			"owner": owner,
			"count": count,
			"total_size": totalSize,
		})
	}
	return stats, nil
}

func (db *DB) GetTemporalStats(ctx context.Context, fileID int64) ([]map[string]interface{}, error) {
	query := `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM metadata_records 
		WHERE file_id = ? AND created_at IS NOT NULL
		GROUP BY DATE(created_at)
		ORDER BY date`
	rows, err := db.conn.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var stats []map[string]interface{}
	for rows.Next() {
		var date string
		var count int64
		if err := rows.Scan(&date, &count); err != nil {
			return nil, err
		}
		stats = append(stats, map[string]interface{}{
			"date": date,
			"count": count,
		})
	}
	return stats, nil
}

// Aggregate statistics
func (db *DB) GetAggregateStats(ctx context.Context, fileID int64) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_files,
			AVG(size) as avg_size,
			MIN(size) as min_size,
			MAX(size) as max_size,
			SUM(size) as total_size,
			COUNT(DISTINCT extension) as unique_types,
			COUNT(DISTINCT owner) as unique_owners
		FROM metadata_records 
		WHERE file_id = ?`
	
	var stats map[string]interface{}
	var totalFiles, uniqueTypes, uniqueOwners int64
	var avgSize, minSize, maxSize, totalSize float64
	err := db.conn.QueryRowContext(ctx, query, fileID).Scan(
		&totalFiles, &avgSize, &minSize, &maxSize, &totalSize, &uniqueTypes, &uniqueOwners,
	)
	if err != nil {
		return nil, err
	}
	stats = map[string]interface{}{
		"total_files": totalFiles,
		"avg_size": avgSize,
		"min_size": minSize,
		"max_size": maxSize,
		"total_size": totalSize,
		"unique_types": uniqueTypes,
		"unique_owners": uniqueOwners,
	}
	return stats, nil
}