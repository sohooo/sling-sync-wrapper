package main

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestProcessLogLineValid(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "run")
	rows, err := processLogLine(`{"level":"info","message":"rows","rows":5}`, span)
	span.End()
	if err != nil {
		t.Fatalf("processLogLine error: %v", err)
	}
	if rows != 5 {
		t.Fatalf("expected 5 rows, got %d", rows)
	}

	ended := sr.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected one span, got %d", len(ended))
	}
	events := ended[0].Events()
	if len(events) != 1 || events[0].Name != "rows" {
		t.Fatalf("expected rows event, got %v", events)
	}
}

func TestProcessLogLineInvalidJSON(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "run")
	if _, err := processLogLine("not json", span); err == nil {
		t.Fatalf("expected error")
	}
	span.End()

	ended := sr.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected one span, got %d", len(ended))
	}
	events := ended[0].Events()
	found := false
	for _, e := range events {
		if e.Name == "invalid JSON log line" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected invalid JSON log line event, got %v", events)
	}
}

func TestProcessLogLineWithErrorField(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "run")
	rows, err := processLogLine(`{"level":"error","message":"fail","error":"boom"}`, span)
	span.End()
	if err != nil {
		t.Fatalf("processLogLine error: %v", err)
	}
	if rows != 0 {
		t.Fatalf("expected 0 rows, got %d", rows)
	}

	ended := sr.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected one span, got %d", len(ended))
	}
	events := ended[0].Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}
