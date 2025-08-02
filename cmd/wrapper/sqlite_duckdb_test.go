package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"sling-sync-wrapper/internal/config"
	"sling-sync-wrapper/internal/sampledb"
)

func TestSQLiteToDuckDBSync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmp := t.TempDir()
	missionPath := filepath.Join(tmp, "mission.db")
	commandPath := filepath.Join(tmp, "command.db")
	pipelinePath := filepath.Join(tmp, "pipeline.yaml")

	if err := sampledb.CreateMissionDB(missionPath, "mission-01", 1); err != nil {
		t.Fatalf("create mission db: %v", err)
	}

	os.WriteFile(pipelinePath, []byte(""), 0644)

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	cfg := config.Config{MissionClusterID: "mc", StateLocation: filepath.Join(tmp, "state"), SyncMode: "normal", MaxRetries: 1, BackoffBase: time.Millisecond}

	runSlingOnceFunc = func(ctx context.Context, bin, pipeline, state, jobID string, span trace.Span) (int, error) {
		if err := sampledb.EnsureCommandTable(commandPath, false); err != nil {
			return 0, err
		}
		return sampledb.Sync(missionPath, commandPath, "", false)
	}
	defer func() { runSlingOnceFunc = runSlingOnce }()

	if err := runPipeline(context.Background(), tracer, cfg, pipelinePath, "job1"); err != nil {
		t.Fatalf("runPipeline returned error: %v", err)
	}

	commandDB, err := sql.Open("duckdb", commandPath)
	if err != nil {
		t.Fatalf("open command: %v", err)
	}
	defer commandDB.Close()

	var cnt int
	if err := commandDB.QueryRow(`select count(*) from telemetry`).Scan(&cnt); err != nil {
		t.Fatalf("query count: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 row, got %d", cnt)
	}
}

func TestSQLiteTwoMissionDBsToDuckDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmp := t.TempDir()

	mission1Path := filepath.Join(tmp, "mission1.db")
	mission2Path := filepath.Join(tmp, "mission2.db")
	commandPath := filepath.Join(tmp, "command.db")

	pipeline1 := filepath.Join(tmp, "pipeline1.yaml")
	pipeline2 := filepath.Join(tmp, "pipeline2.yaml")
	os.WriteFile(pipeline1, []byte(""), 0644)
	os.WriteFile(pipeline2, []byte(""), 0644)

	if err := sampledb.CreateMissionDB(mission1Path, "mission1", 5); err != nil {
		t.Fatalf("create mission1: %v", err)
	}
	if err := sampledb.CreateMissionDB(mission2Path, "mission2", 5); err != nil {
		t.Fatalf("create mission2: %v", err)
	}

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	cfg := config.Config{MissionClusterID: "mc", StateLocation: filepath.Join(tmp, "state"), SyncMode: "normal", MaxRetries: 1, BackoffBase: time.Millisecond}

	var srcPath string
	var currentMission string
	runSlingOnceFunc = func(ctx context.Context, bin, pipeline, state, jobID string, span trace.Span) (int, error) {
		if err := sampledb.EnsureCommandTable(commandPath, true); err != nil {
			return 0, err
		}
		return sampledb.Sync(srcPath, commandPath, currentMission, true)
	}
	defer func() { runSlingOnceFunc = runSlingOnce }()

	srcPath = mission1Path
	currentMission = "mission1"
	if err := runPipeline(context.Background(), tracer, cfg, pipeline1, "job1"); err != nil {
		t.Fatalf("runPipeline returned error: %v", err)
	}
	srcPath = mission2Path
	currentMission = "mission2"
	if err := runPipeline(context.Background(), tracer, cfg, pipeline2, "job2"); err != nil {
		t.Fatalf("runPipeline returned error: %v", err)
	}

	commandDB, err := sql.Open("duckdb", commandPath)
	if err != nil {
		t.Fatalf("open command: %v", err)
	}
	defer commandDB.Close()

	var cnt int
	if err := commandDB.QueryRow(`select count(*) from telemetry`).Scan(&cnt); err != nil {
		t.Fatalf("query count: %v", err)
	}
	if cnt != 10 {
		t.Fatalf("expected 10 rows, got %d", cnt)
	}

	var syncedFrom string
	if err := commandDB.QueryRow(`select synced_from from telemetry limit 1`).Scan(&syncedFrom); err != nil {
		t.Fatalf("synced_from column missing: %v", err)
	}
	if syncedFrom == "" {
		t.Fatalf("synced_from column empty")
	}

	var mission1Count, mission2Count int
	if err := commandDB.QueryRow(`select count(*) from telemetry where synced_from = 'mission1'`).Scan(&mission1Count); err != nil {
		t.Fatalf("count mission1: %v", err)
	}
	if err := commandDB.QueryRow(`select count(*) from telemetry where synced_from = 'mission2'`).Scan(&mission2Count); err != nil {
		t.Fatalf("count mission2: %v", err)
	}
	if mission1Count != 5 || mission2Count != 5 {
		t.Fatalf("unexpected synced_from counts: mission1=%d mission2=%d", mission1Count, mission2Count)
	}
}
