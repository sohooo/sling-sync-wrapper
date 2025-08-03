package config

import (
	"os"
	"path/filepath"
	"reflect"
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
	if cfg.SlingBinary != "sling" {
		t.Errorf("unexpected default sling binary: %s", cfg.SlingBinary)
	}
	if cfg.SlingTimeout != 30*time.Minute {
		t.Errorf("unexpected default sling timeout: %s", cfg.SlingTimeout)
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
	t.Setenv("SLING_BIN", "/usr/local/bin/sling")
	t.Setenv("SLING_TIMEOUT", "10s")

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
	if cfg.SlingBinary != "/usr/local/bin/sling" {
		t.Errorf("unexpected sling binary: %s", cfg.SlingBinary)
	}
	if cfg.SlingTimeout != 10*time.Second {
		t.Errorf("unexpected sling timeout: %s", cfg.SlingTimeout)
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
	expected := []string{f1, f2}
	if !reflect.DeepEqual(files, expected) {
		t.Fatalf("expected %v, got %v", expected, files)
	}
}

func TestPipelinesBothSet(t *testing.T) {
	dir := t.TempDir()
	_, err := Pipelines(Config{PipelineDir: dir, PipelineFile: "p.yaml"})
	if err == nil {
		t.Fatalf("expected error when both PipelineDir and PipelineFile are set")
	}
}

func TestPipelinesMissing(t *testing.T) {
	_, err := Pipelines(Config{})
	if err == nil {
		t.Fatalf("expected error for missing config")
	}
}

func TestPipelinesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := Pipelines(Config{PipelineDir: dir})
	if err == nil {
		t.Fatalf("expected error for empty pipeline dir")
	}
}

func TestGetEnvInt(t *testing.T) {
	const key = "TEST_ENV_INT"

	os.Unsetenv(key)
	if got := getEnvInt(key, 42); got != 42 {
		t.Fatalf("expected fallback 42, got %d", got)
	}

	t.Setenv(key, "7")
	if got := getEnvInt(key, 42); got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}

	t.Setenv(key, "notanint")
	if got := getEnvInt(key, 42); got != 42 {
		t.Fatalf("expected fallback 42 on invalid int, got %d", got)
	}
}
