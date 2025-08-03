package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"sling-sync-wrapper/internal/config"
	"sling-sync-wrapper/internal/logging"
)

func TestRunPipelineNoop(t *testing.T) {
	var called bool
	runSlingOnceFunc = func(ctx context.Context, bin, pipeline, state, jobID string, span trace.Span) (int, error) {
		called = true
		return 0, nil
	}
	defer func() { runSlingOnceFunc = runSlingOnce }()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := logging.NewContext(context.Background(), logger)

	cfg := config.Config{MissionClusterID: "mc", StateLocation: "state", SyncMode: "noop", MaxRetries: 1, BackoffBase: time.Millisecond}
	if err := runPipeline(ctx, tracer, cfg, "pipe.yaml", "job1"); err != nil {
		t.Fatalf("runPipeline returned error: %v", err)
	}

	if called {
		t.Fatalf("runSlingOnce should not be called in noop mode")
	}
	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected one span, got %d", len(spans))
	}
	found := false
	for _, attr := range spans[0].Attributes() {
		if attr.Key == "status" && attr.Value.AsString() == "noop" {
			found = true
		}
	}
	if !found {
		t.Errorf("noop status attribute missing")
	}
	if !bytes.Contains(buf.Bytes(), []byte("\"mode\":\"noop\"")) {
		t.Errorf("noop log not produced")
	}
}

func TestRunPipelineBackfill(t *testing.T) {
	var called bool
	runSlingOnceFunc = func(ctx context.Context, bin, pipeline, state, jobID string, span trace.Span) (int, error) {
		called = true
		return 0, nil
	}
	defer func() { runSlingOnceFunc = runSlingOnce }()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	cfg := config.Config{MissionClusterID: "mc", StateLocation: "state", SyncMode: "backfill", MaxRetries: 1, BackoffBase: time.Millisecond}
	if err := runPipeline(testContext(), tracer, cfg, "pipe.yaml", "job1"); err != nil {
		t.Fatalf("runPipeline returned error: %v", err)
	}

	if called {
		t.Fatalf("runSlingOnce should not be called in backfill mode")
	}

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected one span, got %d", len(spans))
	}
	found := false
	for _, attr := range spans[0].Attributes() {
		if attr.Key == "status" && attr.Value.AsString() == "backfill" {
			found = true
		}
	}
	if !found {
		t.Errorf("backfill status attribute missing")
	}
}

func TestRunPipelineReturnsError(t *testing.T) {
	runSlingOnceFunc = func(ctx context.Context, bin, pipeline, state, jobID string, span trace.Span) (int, error) {
		return 0, fmt.Errorf("boom")
	}
	defer func() { runSlingOnceFunc = runSlingOnce }()

	tracer := trace.NewNoopTracerProvider().Tracer("test")
	cfg := config.Config{MissionClusterID: "mc", StateLocation: "state", SyncMode: "normal", MaxRetries: 2, BackoffBase: time.Millisecond}
	if err := runPipeline(testContext(), tracer, cfg, "pipe.yaml", "job1"); err == nil {
		t.Fatalf("expected error from runPipeline")
	}
}
