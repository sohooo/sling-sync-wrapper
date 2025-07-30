package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func fakeExecCommandContext(script string) func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, command string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, script)
	}
}

func writeScript(dir string) (string, error) {
	script := filepath.Join(dir, "sling")
	content := "#!/bin/sh\n" +
		"echo '{\"level\":\"info\",\"message\":\"start\"}'\n" +
		"echo '{\"level\":\"info\",\"message\":\"rows\",\"rows\":10}'\n" +
		"echo '{\"level\":\"error\",\"message\":\"fail\",\"error\":\"boom\"}'\n"
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		return "", err
	}
	return script, nil
}

func TestRunSlingOnceEvents(t *testing.T) {
	dir := t.TempDir()
	script, err := writeScript(dir)
	if err != nil {
		t.Fatalf("script: %v", err)
	}
	execCommandContext = fakeExecCommandContext(script)
	defer func() { execCommandContext = exec.CommandContext }()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "run")
	rows, err := runSlingOnce(ctx, script, "pipe.yaml", "state", "job", span)
	span.End()
	if err != nil {
		t.Fatalf("runSlingOnce error: %v", err)
	}
	if rows != 10 {
		t.Fatalf("expected 10 rows, got %d", rows)
	}

	ended := sr.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected one span, got %d", len(ended))
	}
	if len(ended[0].Events()) < 3 {
		t.Errorf("expected at least 3 events, got %d", len(ended[0].Events()))
	}
}
