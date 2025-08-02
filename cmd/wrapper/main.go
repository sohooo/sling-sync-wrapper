package main

import (
	"context"
	"log"
	"os"
	"time"

	"net/url"
	"path/filepath"

	"github.com/google/uuid"

	"sling-sync-wrapper/internal/config"
	"sling-sync-wrapper/internal/tracing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	runSlingOnceFunc = runSlingOnce
	sleepFunc        = time.Sleep
	removeAllFunc    = os.RemoveAll
)

func main() {
	ctx := context.Background()

	cfg := config.FromEnv()

	pipelines, err := config.Pipelines(cfg)
	if err != nil {
		log.Fatal(err)
	}

	tracer, shutdown := tracing.Init(ctx, "sling-sync-wrapper", cfg.MissionClusterID, cfg.OTELEndpoint)
	defer shutdown(ctx)

	var failed bool
	for _, pipeline := range pipelines {
		jobID := uuid.NewString()
		if err := runPipeline(ctx, tracer, cfg, pipeline, jobID); err != nil {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func runPipeline(ctx context.Context, tracer trace.Tracer, cfg config.Config, pipeline, jobID string) error {
	ctx, span := tracer.Start(ctx, "sling.sync.run")
	defer span.End()

	span.SetAttributes(
		attribute.String("mission_cluster_id", cfg.MissionClusterID),
		attribute.String("sync_job_id", jobID),
		attribute.String("pipeline", pipeline),
		attribute.String("state_location", cfg.StateLocation),
		attribute.String("sync_mode", cfg.SyncMode),
	)

	startTime := time.Now()

	if cfg.SyncMode == "noop" {
		log.Printf("[NOOP] Would run Sling pipeline %s", pipeline)
		span.SetAttributes(attribute.String("status", "noop"))
		return nil
	}

	if cfg.SyncMode == "backfill" {
		log.Printf("[BACKFILL] Resetting sync state at %s", cfg.StateLocation)
		u, err := url.Parse(cfg.StateLocation)
		if err != nil {
			log.Printf("Invalid state location: %v", err)
		} else if u.Scheme != "" && u.Scheme != "file" {
			log.Printf("State location scheme %q is not supported for backfill", u.Scheme)
		} else {
			p := u.Path
			if p == "" {
				p = u.Opaque
			}
			p = filepath.Clean(p)
			if p == "." || p == string(os.PathSeparator) {
				log.Printf("State location path %q is unsafe; skipping reset", p)
			} else if err := removeAllFunc(p); err != nil {
				log.Printf("Failed to reset state: %v", err)
			}
		}
	}

	var lastErr error
	var rowsSynced int
	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
		rows, err := runSlingOnceFunc(ctx, cfg.SlingBinary, pipeline, cfg.StateLocation, jobID, span)
		rowsSynced += rows
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
		wait := cfg.BackoffBase * time.Duration(1<<uint(attempt-1))
		log.Printf("Attempt %d failed: %v, retrying in %s", attempt, err, wait)
		sleepFunc(wait)
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

	log.Printf("Pipeline %s completed in %.2fs (rows: %d, status: %s)", pipeline, duration.Seconds(), rowsSynced, statusFromErr(lastErr))
	return lastErr
}
