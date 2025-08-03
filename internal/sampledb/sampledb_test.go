package sampledb

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
)

func TestEnsureCommandTableSyncedFromColumn(t *testing.T) {
	tmp := t.TempDir()

	without := filepath.Join(tmp, "without.db")
	if err := EnsureCommandTable(without, false); err != nil {
		t.Fatalf("ensure without synced_from: %v", err)
	}
	db, err := sql.Open("duckdb", without)
	if err != nil {
		t.Fatalf("open without: %v", err)
	}
	defer db.Close()
	var cnt int
	if err := db.QueryRow(`select count(*) from information_schema.columns where table_name='telemetry' and column_name='synced_from'`).Scan(&cnt); err != nil {
		t.Fatalf("query without synced_from: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("unexpected synced_from column in table")
	}

	with := filepath.Join(tmp, "with.db")
	if err := EnsureCommandTable(with, true); err != nil {
		t.Fatalf("ensure with synced_from: %v", err)
	}
	db2, err := sql.Open("duckdb", with)
	if err != nil {
		t.Fatalf("open with: %v", err)
	}
	defer db2.Close()
	if err := db2.QueryRow(`select count(*) from information_schema.columns where table_name='telemetry' and column_name='synced_from'`).Scan(&cnt); err != nil {
		t.Fatalf("query with synced_from: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("synced_from column missing")
	}
}

func TestSyncCopiesRows(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.db")
	if err := CreateMissionDB(src, "m1", 2); err != nil {
		t.Fatalf("create mission: %v", err)
	}

	dst1 := filepath.Join(tmp, "dst1.db")
	if err := EnsureCommandTable(dst1, false); err != nil {
		t.Fatalf("ensure dst1: %v", err)
	}
	if _, err := Sync(src, dst1, "m1", false); err != nil {
		t.Fatalf("sync dst1: %v", err)
	}
	db1, err := sql.Open("duckdb", dst1)
	if err != nil {
		t.Fatalf("open dst1: %v", err)
	}
	defer db1.Close()
	var cnt int
	if err := db1.QueryRow(`select count(*) from telemetry`).Scan(&cnt); err != nil {
		t.Fatalf("count dst1: %v", err)
	}
	if cnt != 2 {
		t.Fatalf("expected 2 rows in dst1, got %d", cnt)
	}

	dst2 := filepath.Join(tmp, "dst2.db")
	if err := EnsureCommandTable(dst2, true); err != nil {
		t.Fatalf("ensure dst2: %v", err)
	}
	if _, err := Sync(src, dst2, "mission1", true); err != nil {
		t.Fatalf("sync dst2: %v", err)
	}
	db2, err := sql.Open("duckdb", dst2)
	if err != nil {
		t.Fatalf("open dst2: %v", err)
	}
	defer db2.Close()
	if err := db2.QueryRow(`select count(*) from telemetry`).Scan(&cnt); err != nil {
		t.Fatalf("count dst2: %v", err)
	}
	if cnt != 2 {
		t.Fatalf("expected 2 rows in dst2, got %d", cnt)
	}
	var syncedFrom string
	if err := db2.QueryRow(`select synced_from from telemetry limit 1`).Scan(&syncedFrom); err != nil {
		t.Fatalf("query synced_from: %v", err)
	}
	if syncedFrom != "mission1" {
		t.Fatalf("unexpected synced_from value %q", syncedFrom)
	}
}
