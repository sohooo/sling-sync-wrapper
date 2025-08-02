package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sling-sync-wrapper/internal/config"

	"go.opentelemetry.io/otel/trace"
)

func TestRunPipelineBackfillRemovesFileState(t *testing.T) {
	var removed string
	removeAllFunc = func(path string) error {
		removed = path
		return nil
	}
	defer func() { removeAllFunc = os.RemoveAll }()

	runSlingOnceFunc = func(ctx context.Context, bin, pipeline, state, jobID string, span trace.Span) (int, error) {
		return 0, nil
	}
	defer func() { runSlingOnceFunc = runSlingOnce }()

	tracer := trace.NewNoopTracerProvider().Tracer("test")

	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	cfg := config.Config{MissionClusterID: "mc", StateLocation: "file://" + stateFile, SyncMode: "backfill", MaxRetries: 1, BackoffBase: time.Millisecond}
	if err := runPipeline(context.Background(), tracer, cfg, "pipe.yaml", "job1"); err != nil {
		t.Fatalf("runPipeline returned error: %v", err)
	}
	if removed != stateFile {
		t.Fatalf("removeAll called with %q, want %q", removed, stateFile)
	}
}

func TestRunPipelineBackfillSkipsNonFileScheme(t *testing.T) {
	var called bool
	removeAllFunc = func(path string) error {
		called = true
		return nil
	}
	defer func() { removeAllFunc = os.RemoveAll }()

	runSlingOnceFunc = func(ctx context.Context, bin, pipeline, state, jobID string, span trace.Span) (int, error) {
		return 0, nil
	}
	defer func() { runSlingOnceFunc = runSlingOnce }()

	tracer := trace.NewNoopTracerProvider().Tracer("test")
	cfg := config.Config{MissionClusterID: "mc", StateLocation: "s3://bucket/state.json", SyncMode: "backfill", MaxRetries: 1, BackoffBase: time.Millisecond}
	if err := runPipeline(context.Background(), tracer, cfg, "pipe.yaml", "job1"); err != nil {
		t.Fatalf("runPipeline returned error: %v", err)
	}
	if called {
		t.Fatalf("removeAll should not be called for non-file scheme")
	}
}
