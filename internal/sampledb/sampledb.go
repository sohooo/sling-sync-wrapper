package sampledb

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/marcboeker/go-duckdb"
	_ "github.com/mattn/go-sqlite3"
)

// CreateMissionDB creates a SQLite database with sample telemetry rows.
func CreateMissionDB(path, cluster string, rows int) error {
	os.Remove(path)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	defer db.Close()

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
		return err
	}

	for i := 0; i < rows; i++ {
		_, err := db.Exec(`insert into telemetry values (?,?,?,?,?,?,?,?)`,
			cluster, fmt.Sprintf("drone-%s-%d", cluster, i),
			48.0+float64(i), 16.0+float64(i), 100.0+float64(i),
			95.0, "ok", fmt.Sprintf("2025-07-23T12:34:%02dZ", i))
		if err != nil {
			return err
		}
	}
	return nil
}

// EnsureCommandTable creates the DuckDB telemetry table if it does not exist.
// If includeSyncedFrom is true, a synced_from column is added.
func EnsureCommandTable(path string, includeSyncedFrom bool) error {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return err
	}
	defer db.Close()

	var b strings.Builder
	b.WriteString(`create table if not exists telemetry (
        cluster_id text,
        drone_id text,
        lat real,
        lon real,
        alt real,
        battery real,
        status text,
        ts text`)
	if includeSyncedFrom {
		b.WriteString(`,
        synced_from text`)
	}
	b.WriteString(`
    )`)

	_, err = db.Exec(b.String())
	return err
}

// Sync copies telemetry rows from a SQLite source to a DuckDB destination.
// mission is written to the synced_from column when includeSyncedFrom is true.
func Sync(srcPath, dstPath, mission string, includeSyncedFrom bool) (int, error) {
	src, err := sql.Open("sqlite3", srcPath)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	dst, err := sql.Open("duckdb", dstPath)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	rows, err := src.Query(`select cluster_id, drone_id, lat, lon, alt, battery, status, ts from telemetry`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var b strings.Builder
	b.WriteString(`insert into telemetry values (?,?,?,?,?,?,?,?`)
	if includeSyncedFrom {
		b.WriteString(",?")
	}
	b.WriteString(")")

	stmt, err := dst.Prepare(b.String())
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	count := 0
	for rows.Next() {
		var cID, dID, status, ts string
		var lat, lon, alt, battery float64
		if err := rows.Scan(&cID, &dID, &lat, &lon, &alt, &battery, &status, &ts); err != nil {
			return count, err
		}
		args := []any{cID, dID, lat, lon, alt, battery, status, ts}
		if includeSyncedFrom {
			args = append(args, mission)
		}
		if _, err := stmt.Exec(args...); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}

// CountRows returns the number of rows in the DuckDB telemetry table.
func CountRows(path string) (int, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var cnt int
	if err := db.QueryRow(`select count(*) from telemetry`).Scan(&cnt); err != nil {
		return 0, err
	}
	return cnt, nil
}
