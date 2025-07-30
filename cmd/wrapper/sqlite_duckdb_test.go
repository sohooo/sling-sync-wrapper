package main

import (
	"context"
	"database/sql"
	"fmt"
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

	runSlingOnceFunc = func(ctx context.Context, bin, pipeline, state, jobID string, span trace.Span) (int, error) {
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

	createMissionDB := func(path, cluster string) *sql.DB {
		db, err := sql.Open("sqlite3", path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		t.Cleanup(func() { db.Close() })

		_, err = db.Exec(`create table telemetry (
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

		for i := 0; i < 5; i++ {
			_, err := db.Exec(`insert into telemetry values (?,?,?,?,?,?,?,?)`,
				cluster, fmt.Sprintf("drone-%s-%d", cluster, i),
				48.0+float64(i), 16.0+float64(i), 100.0+float64(i),
				95.0, "ok", fmt.Sprintf("2025-07-23T12:34:%02dZ", i))
			if err != nil {
				t.Fatalf("insert: %v", err)
			}
		}
		return db
	}

	createMissionDB(mission1Path, "mission1")
	createMissionDB(mission2Path, "mission2")

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	cfg := config.Config{MissionClusterID: "mc", StateLocation: filepath.Join(tmp, "state"), SyncMode: "normal", MaxRetries: 1, BackoffBase: time.Millisecond}

	var srcPath string
	var currentMission string
	runSlingOnceFunc = func(ctx context.Context, bin, pipeline, state, jobID string, span trace.Span) (int, error) {
		src, err := sql.Open("sqlite3", srcPath)
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
            ts text,
            synced_from text
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
			if _, err := dst.Exec(`insert into telemetry values (?,?,?,?,?,?,?,?,?)`,
				cID, dID, lat, lon, alt, battery, status, ts, currentMission); err != nil {
				return count, err
			}
			count++
		}
		return count, rows.Err()
	}
	defer func() { runSlingOnceFunc = runSlingOnce }()

	srcPath = mission1Path
	currentMission = "mission1"
	runPipeline(context.Background(), tracer, cfg, pipeline1, "job1")
	srcPath = mission2Path
	currentMission = "mission2"
	runPipeline(context.Background(), tracer, cfg, pipeline2, "job2")

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
