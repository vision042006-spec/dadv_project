package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Worker   WorkerConfig
	Security SecurityConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Driver string
	DSN    string
}

type RedisConfig struct {
	Addr         string
	Password    string
	DB           int
	PoolSize     int
	MinIdleConns int
}

type WorkerConfig struct {
	QueueName      string
	BatchSize      int
	ProcessTimeout time.Duration
}

type SecurityConfig struct {
	AllowedFileTypes   []string
	MaxFileSize        int64
	RateLimitRequests  int
	RateLimitDuration  time.Duration
	CORSAllowedOrigins []string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("API_PORT", "8080"),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Database: DatabaseConfig{
			Driver: "sqlite3",
			DSN:    getEnv("DATABASE_DSN", "./data/dadv.db"),
		},
		Redis: RedisConfig{
			Addr:         getEnv("REDIS_ADDR", "localhost:6379"),
			Password:    getEnv("REDIS_PASSWORD", ""),
			DB:          getEnvInt("REDIS_DB", 0),
			PoolSize:     10,
			MinIdleConns: 2,
		},
		Worker: WorkerConfig{
			QueueName:      "metadata_jobs",
			BatchSize:     100,
			ProcessTimeout: 5 * time.Minute,
		},
		Security: SecurityConfig{
			AllowedFileTypes: []string{
				"text/csv",
				"application/json",
				"application/vnd.ms-excel",
				"application/x-excel",
				"application/x-msexcel",
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			},
			MaxFileSize:        100 * 1024 * 1024, // 100MB
			RateLimitRequests: 100,
			RateLimitDuration:  time.Minute,
			CORSAllowedOrigins: getCORSOrigins(),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getCORSOrigins() []string {
	defaults := []string{
		"http://localhost:5173",
		"http://localhost:3000",
	}
	extra := os.Getenv("CORS_ALLOWED_ORIGINS")
	if extra == "" {
		return defaults
	}
	var origins []string
	for _, o := range append(defaults, splitCSV(extra)...) {
		origins = append(origins, o)
	}
	return origins
}

func splitCSV(s string) []string {
	var parts []string
	for _, p := range stringSplit(s, ",") {
		if trimmed := stringTrimSpace(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func stringSplit(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func stringTrimSpace(s string) string {
	start, end := 0, len(s)-1
	for start <= end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end >= start && (s[end] == ' ' || s[end] == '\t') {
		end--
	}
	if start > end {
		return ""
	}
	return s[start : end+1]
}