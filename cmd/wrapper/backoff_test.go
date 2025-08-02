package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"sling-sync-wrapper/internal/config"

	"go.opentelemetry.io/otel/trace"
)

func TestRunPipelineExponentialBackoff(t *testing.T) {
	var calls int
	runSlingOnceFunc = func(ctx context.Context, bin, pipeline, state, jobID string, span trace.Span) (int, error) {
		calls++
		if calls < 4 {
			return 0, fmt.Errorf("fail %d", calls)
		}
		return 0, nil
	}
	defer func() { runSlingOnceFunc = runSlingOnce }()

	var sleeps []time.Duration
	sleepFunc = func(d time.Duration) {
		sleeps = append(sleeps, d)
	}
	defer func() { sleepFunc = time.Sleep }()

	tracer := trace.NewNoopTracerProvider().Tracer("test")
	cfg := config.Config{MissionClusterID: "mc", StateLocation: "state", SyncMode: "normal", MaxRetries: 4, BackoffBase: time.Millisecond}

	runPipeline(context.Background(), tracer, cfg, "pipe.yaml", "job1")

	expected := []time.Duration{cfg.BackoffBase, 2 * cfg.BackoffBase, 4 * cfg.BackoffBase}
	if len(sleeps) != len(expected) {
		t.Fatalf("expected %d sleeps, got %d", len(expected), len(sleeps))
	}
	for i, d := range expected {
		if sleeps[i] != d {
			t.Fatalf("sleep %d = %v, want %v", i, sleeps[i], d)
		}
	}
}
