package config

import (
	"fmt"
	"os"
	"path/filepath"
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
	}
}

// Pipelines returns a list of pipeline files to run.
func Pipelines(cfg Config) ([]string, error) {
	var pipelines []string
	if cfg.PipelineDir != "" {
		files, err := filepath.Glob(filepath.Join(cfg.PipelineDir, "*.yaml"))
		if err != nil {
			return nil, err
		}
		pipelines = append(pipelines, files...)
	} else if cfg.PipelineFile != "" {
		pipelines = append(pipelines, cfg.PipelineFile)
	} else {
		return nil, fmt.Errorf("no SLING_CONFIG or PIPELINE_DIR specified")
	}
	return pipelines, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		fmt.Sscanf(v, "%d", &i)
		return i
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
