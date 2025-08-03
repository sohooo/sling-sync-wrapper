package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

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

// run executes all configured pipelines according to cfg.
func run(ctx context.Context, cfg config.Config) error {
	pipelines, err := config.Pipelines(cfg)
	if err != nil {
		return fmt.Errorf("load pipelines: %w", err)
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
		return fmt.Errorf("one or more pipelines failed")
	}
	return nil
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

	switch cfg.SyncMode {
	case "noop":
		log.Printf("[NOOP] Would run Sling pipeline %s", pipeline)
		span.SetAttributes(attribute.String("status", "noop"))
		return nil
	case "backfill":
		if err := resetState(cfg); err != nil {
			span.RecordError(err)
			span.SetAttributes(attribute.String("status", "failed"))
			return fmt.Errorf("reset state: %w", err)
		}
		span.SetAttributes(attribute.String("status", "backfill"))
		return nil
	}

	startTime := time.Now()

	prevTimeout := slingCLITimeout
	slingCLITimeout = cfg.SlingTimeout
	defer func() { slingCLITimeout = prevTimeout }()

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
	if lastErr != nil {
		return fmt.Errorf("sling run failed: %w", lastErr)
	}
	return nil
}
