package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	MissionClusterID string
	PipelineFile     string
	PipelineDir      string
	StateLocation    string
	OTELEndpoint     string
	SyncMode         string
	MaxRetries       int
	BackoffBase      time.Duration
	SlingBinary      string
	SlingTimeout     time.Duration
}

// FromEnv constructs a Config from environment variables.
func FromEnv() Config {
	return Config{
		MissionClusterID: getEnv("MISSION_CLUSTER_ID", "unknown-cluster"),
		PipelineFile:     os.Getenv("SLING_CONFIG"),
		PipelineDir:      os.Getenv("PIPELINE_DIR"),
		StateLocation:    getEnv("SLING_STATE", "file://./sling_state.json"),
		OTELEndpoint:     getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		SyncMode:         getEnv("SYNC_MODE", "normal"),
		MaxRetries:       getEnvInt("SYNC_MAX_RETRIES", 3),
		BackoffBase:      getEnvDuration("SYNC_BACKOFF_BASE", 5*time.Second),
		SlingBinary:      getEnv("SLING_BIN", "sling"),
		SlingTimeout:     getEnvDuration("SLING_TIMEOUT", 30*time.Minute),
	}
}

// Pipelines returns a list of pipeline files to run.
func Pipelines(cfg Config) ([]string, error) {
	if cfg.PipelineDir != "" && cfg.PipelineFile != "" {
		return nil, fmt.Errorf("cannot set both PipelineDir and PipelineFile")
	}

	switch {
	case cfg.PipelineDir != "":
		files, err := filepath.Glob(filepath.Join(cfg.PipelineDir, "*.yaml"))
		if err != nil {
			return nil, fmt.Errorf("find pipeline files: %w", err)
		}
		sort.Strings(files)
		if len(files) > 0 {
			return files, nil
		}
	case cfg.PipelineFile != "":
		return []string{cfg.PipelineFile}, nil
	}

	return nil, fmt.Errorf("no pipeline files found (set SLING_CONFIG or PIPELINE_DIR)")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
