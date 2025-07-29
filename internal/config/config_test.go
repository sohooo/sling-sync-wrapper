package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFromEnvDefaults(t *testing.T) {
	os.Clearenv()
	cfg := FromEnv()
	if cfg.MissionClusterID != "unknown-cluster" {
		t.Errorf("expected default MissionClusterID, got %s", cfg.MissionClusterID)
	}
	if cfg.StateLocation != "file://./sling_state.json" {
		t.Errorf("unexpected default state location: %s", cfg.StateLocation)
	}
	if cfg.OTELEndpoint != "localhost:4317" {
		t.Errorf("unexpected default OTEL endpoint: %s", cfg.OTELEndpoint)
	}
	if cfg.SyncMode != "normal" {
		t.Errorf("unexpected default sync mode: %s", cfg.SyncMode)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("unexpected default retries: %d", cfg.MaxRetries)
	}
	if cfg.BackoffBase != 5*time.Second {
		t.Errorf("unexpected default backoff: %s", cfg.BackoffBase)
	}
}

func TestFromEnvOverrides(t *testing.T) {
	t.Setenv("MISSION_CLUSTER_ID", "mc1")
	t.Setenv("SLING_CONFIG", "pipeline.yaml")
	t.Setenv("SLING_STATE", "state.json")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel:4317")
	t.Setenv("SYNC_MODE", "backfill")
	t.Setenv("SYNC_MAX_RETRIES", "5")
	t.Setenv("SYNC_BACKOFF_BASE", "2s")

	cfg := FromEnv()
	if cfg.MissionClusterID != "mc1" || cfg.PipelineFile != "pipeline.yaml" {
		t.Errorf("unexpected cfg: %+v", cfg)
	}
	if cfg.StateLocation != "state.json" || cfg.SyncMode != "backfill" {
		t.Errorf("unexpected cfg values: %+v", cfg)
	}
	if cfg.MaxRetries != 5 || cfg.BackoffBase != 2*time.Second {
		t.Errorf("unexpected retry/backoff: %+v", cfg)
	}
	if cfg.OTELEndpoint != "otel:4317" {
		t.Errorf("unexpected otel endpoint: %s", cfg.OTELEndpoint)
	}
}

func TestPipelinesFile(t *testing.T) {
	cfg := Config{PipelineFile: "p1.yaml"}
	files, err := Pipelines(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "p1.yaml" {
		t.Errorf("unexpected files: %v", files)
	}
}

func TestPipelinesDir(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.yaml")
	f2 := filepath.Join(dir, "b.yaml")
	os.WriteFile(f1, []byte("a"), 0644)
	os.WriteFile(f2, []byte("b"), 0644)

	files, err := Pipelines(Config{PipelineDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %v", files)
	}
}

func TestPipelinesMissing(t *testing.T) {
	_, err := Pipelines(Config{})
	if err == nil {
		t.Fatalf("expected error for missing config")
	}
}
