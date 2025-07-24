package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type SlingLogLine struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Rows    int    `json:"rows,omitempty"`
	Error   string `json:"error,omitempty"`
}

// getEnv fetches an environment variable or returns a fallback value
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	ctx := context.Background()

	// --- Env variables ---
	missionClusterID := getEnv("MISSION_CLUSTER_ID", "unknown-cluster")
	pipelineFile := os.Getenv("SLING_CONFIG")
	pipelineDir := os.Getenv("PIPELINE_DIR") // directory support
	stateLocation := getEnv("SLING_STATE", "file://./sling_state.json")
	otelEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	syncMode := getEnv("SYNC_MODE", "normal") // normal, noop, backfill
	maxRetries := getEnvInt("SYNC_MAX_RETRIES", 3)
	backoffBase := getEnvDuration("SYNC_BACKOFF_BASE", 5*time.Second)

	// --- Init OTel ---
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelEndpoint))
	if err != nil {
		log.Fatalf("failed to create OTLP trace exporter: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("sling-sync-wrapper"),
			attribute.String("mission_cluster_id", missionClusterID),
		)),
	)
	defer tp.Shutdown(ctx)
	otel.SetTracerProvider(tp)
	tracer := otel.Tracer("sling-sync-wrapper")

	// --- Choose pipelines (single file or directory) ---
	var pipelines []string
	if pipelineDir != "" {
		files, _ := filepath.Glob(filepath.Join(pipelineDir, "*.yaml"))
		pipelines = append(pipelines, files...)
	} else if pipelineFile != "" {
		pipelines = append(pipelines, pipelineFile)
	} else {
		log.Fatal("No SLING_CONFIG or PIPELINE_DIR specified")
	}

	for _, pipeline := range pipelines {
		jobID := uuid.NewString()
		os.Setenv("SYNC_JOB_ID", jobID)
		os.Setenv("SLING_CONFIG", pipeline)
		runPipeline(ctx, tracer, missionClusterID, pipeline, stateLocation, jobID, syncMode, maxRetries, backoffBase)
	}
}

func runPipeline(ctx context.Context, tracer sdktrace.Tracer, missionClusterID, pipeline, stateLocation, jobID, syncMode string, maxRetries int, backoffBase time.Duration) {
	ctx, span := tracer.Start(ctx, "sling.sync.run")
	span.SetAttributes(
		attribute.String("mission_cluster_id", missionClusterID),
		attribute.String("sync_job_id", jobID),
		attribute.String("pipeline", pipeline),
		attribute.String("state_location", stateLocation),
		attribute.String("sync_mode", syncMode),
	)

	startTime := time.Now()

	// Noop mode
	if syncMode == "noop" {
		log.Printf("[NOOP] Would run Sling pipeline %s", pipeline)
		span.SetAttributes(attribute.String("status", "noop"))
		span.End()
		return
	}

	// Backfill mode → reset state
	if syncMode == "backfill" {
		log.Printf("[BACKFILL] Resetting sync state at %s", stateLocation)
		if err := os.RemoveAll(stateLocation); err != nil {
			log.Printf("Failed to reset state: %v", err)
		}
	}

	var lastErr error
	var rowsSynced int
	for attempt := 1; attempt <= maxRetries; attempt++ {
		rows, err := runSlingOnce(ctx, pipeline, stateLocation, jobID, span)
		rowsSynced += rows
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
		wait := time.Duration(attempt) * backoffBase
		log.Printf("Attempt %d failed: %v, retrying in %s", attempt, err, wait)
		time.Sleep(wait)
	}

	duration := time.Since(startTime)
	span.SetAttributes(
		attribute.Int("rows_synced", rowsSynced),
		attribute.Float64("duration_seconds", duration.Seconds()),
	)
	if lastErr != nil {
		span.RecordError(lastErr)
		span.SetAttributes(attribute.String("status", "failed"))
	} else {
		span.SetAttributes(attribute.String("status", "success"))
	}
	span.End()
	log.Printf("Pipeline %s completed in %.2fs (rows: %d, status: %s)", pipeline, duration.Seconds(), rowsSynced, statusFromErr(lastErr))
}

func runSlingOnce(ctx context.Context, pipeline, stateLocation, jobID string, span sdktrace.Span) (int, error) {
	cmd := exec.CommandContext(ctx, "sling", "sync", "--config", pipeline, "--log-format", "json")
	cmd.Env = append(os.Environ(), fmt.Sprintf("SLING_STATE=%s", stateLocation))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(stdout)
	rowsSynced := 0
	for scanner.Scan() {
		line := scanner.Text()
		var logEntry SlingLogLine
		if err := json.Unmarshal([]byte(line), &logEntry); err == nil {
			span.AddEvent(logEntry.Message,
				sdktrace.WithAttributes(attribute.String("log.level", logEntry.Level)))
			if logEntry.Rows > 0 {
				rowsSynced += logEntry.Rows
			}
			if logEntry.Error != "" {
				span.RecordError(fmt.Errorf(logEntry.Error))
			}
		} else {
			span.AddEvent(line)
		}
	}

	if err := cmd.Wait(); err != nil {
		return rowsSynced, err
	}
	return rowsSynced, nil
}

func statusFromErr(err error) string {
	if err != nil {
		return "failed"
	}
	return "success"
}

// Helpers for environment conversion
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
