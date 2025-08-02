package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestRunSlingOnceLongLogLine(t *testing.T) {
	dir := t.TempDir()
	longMsg := strings.Repeat("a", 70*1024)
	script := filepath.Join(dir, "sling")
	content := fmt.Sprintf("#!/bin/sh\necho '{\"level\":\"info\",\"message\":\"%s\"}'\n", longMsg)
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatalf("script: %v", err)
	}
	execCommandContext = fakeExecCommandContext(script)
	defer func() { execCommandContext = exec.CommandContext }()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "run")
	if _, err := runSlingOnce(ctx, script, "pipe.yaml", "state", "job", span); err != nil {
		t.Fatalf("runSlingOnce error: %v", err)
	}
	span.End()

	ended := sr.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected one span, got %d", len(ended))
	}
	events := ended[0].Events()
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	if got := events[0].Name; len(got) != len(longMsg) {
		t.Fatalf("event truncated: expected len %d, got %d", len(longMsg), len(got))
	}
}

func TestRunSlingOnceEnvironmentVariables(t *testing.T) {
	dir := t.TempDir()
	script, err := writeScript(dir)
	if err != nil {
		t.Fatalf("script: %v", err)
	}

	var capturedCmd *exec.Cmd
	execCommandContext = func(ctx context.Context, command string, args ...string) *exec.Cmd {
		capturedCmd = exec.CommandContext(ctx, script)
		return capturedCmd
	}
	defer func() { execCommandContext = exec.CommandContext }()

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "run")
	if _, err := runSlingOnce(ctx, script, "pipe.yaml", "state", "job", span); err != nil {
		t.Fatalf("runSlingOnce error: %v", err)
	}
	span.End()

	if capturedCmd == nil {
		t.Fatal("execCommandContext was not called")
	}

	env := capturedCmd.Env
	want := []string{
		"SLING_STATE=state",
		"SYNC_JOB_ID=job",
		"SLING_CONFIG=pipe.yaml",
	}
	for _, w := range want {
		found := false
		for _, e := range env {
			if e == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected environment variable %q not found", w)
		}
	}
}
