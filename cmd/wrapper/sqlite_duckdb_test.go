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

	_ "github.com/marcboeker/go-duckdb"
	_ "github.com/mattn/go-sqlite3"
)

func TestSQLiteToDuckDBSync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmp := t.TempDir()
	missionPath := filepath.Join(tmp, "mission.db")
	commandPath := filepath.Join(tmp, "command.db")
	pipelinePath := filepath.Join(tmp, "pipeline.yaml")

	mission, err := sql.Open("sqlite3", missionPath)
	if err != nil {
		t.Fatalf("open mission: %v", err)
	}
	defer mission.Close()

	_, err = mission.Exec(`create table telemetry (
        cluster_id text,
        drone_id text,
        lat real,
        lon real,
        alt real,
        battery real,
        status text,
        ts text
    )`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = mission.Exec(`insert into telemetry values (?,?,?,?,?,?,?,?)`,
		"mission-01", "recon-swarm-123456-A", 48.2023, 16.4098, 100.5, 99.5, "ok", "2025-07-23T12:34:56Z")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	os.WriteFile(pipelinePath, []byte(""), 0644)

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	cfg := config.Config{MissionClusterID: "mc", StateLocation: filepath.Join(tmp, "state"), SyncMode: "normal", MaxRetries: 1, BackoffBase: time.Millisecond}

	runSlingOnceFunc = func(ctx context.Context, pipeline, state, jobID string, span trace.Span) (int, error) {
		src, err := sql.Open("sqlite3", missionPath)
		if err != nil {
			return 0, err
		}
		defer src.Close()

		dst, err := sql.Open("duckdb", commandPath)
		if err != nil {
			return 0, err
		}
		defer dst.Close()

		_, err = dst.Exec(`create table if not exists telemetry (
            cluster_id text,
            drone_id text,
            lat real,
            lon real,
            alt real,
            battery real,
            status text,
            ts text
        )`)
		if err != nil {
			return 0, err
		}

		rows, err := src.Query(`select cluster_id, drone_id, lat, lon, alt, battery, status, ts from telemetry`)
		if err != nil {
			return 0, err
		}
		defer rows.Close()

		count := 0
		for rows.Next() {
			var cID, dID, status, ts string
			var lat, lon, alt, battery float64
			if err := rows.Scan(&cID, &dID, &lat, &lon, &alt, &battery, &status, &ts); err != nil {
				return count, err
			}
			if _, err := dst.Exec(`insert into telemetry values (?,?,?,?,?,?,?,?)`, cID, dID, lat, lon, alt, battery, status, ts); err != nil {
				return count, err
			}
			count++
		}
		return count, rows.Err()
	}
	defer func() { runSlingOnceFunc = runSlingOnce }()

	runPipeline(context.Background(), tracer, cfg, pipelinePath, "job1")

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
