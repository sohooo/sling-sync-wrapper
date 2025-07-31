package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	_ "github.com/mattn/go-sqlite3"
)

// Telemetry represents a single drone telemetry record.
type Telemetry struct {
	ClusterID string
	DroneID   string
	Lat       float64
	Lon       float64
	Alt       float64
	Battery   float64
	Status    string
	TS        time.Time
}

func main() {
	rand.Seed(time.Now().UnixNano())

	dir := "quickstart"
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal(err)
	}

	mission1 := filepath.Join(dir, "mission1.db")
	mission2 := filepath.Join(dir, "mission2.db")
	command := filepath.Join(dir, "command.db")

	if err := createMissionDB(mission1, "mission1", 10); err != nil {
		log.Fatalf("setup mission1: %v", err)
	}
	if err := createMissionDB(mission2, "mission2", 10); err != nil {
		log.Fatalf("setup mission2: %v", err)
	}
	if err := ensureCommandTable(command); err != nil {
		log.Fatalf("setup command db: %v", err)
	}

	if err := sync(mission1, command, "mission1"); err != nil {
		log.Fatal(err)
	}
	if err := sync(mission2, command, "mission2"); err != nil {
		log.Fatal(err)
	}

	cnt, err := countRows(command)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Quickstart complete! %d rows synced into %s\n", cnt, command)
}

func createMissionDB(path, cluster string, rows int) error {
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
		tel := Telemetry{
			ClusterID: cluster,
			DroneID:   fmt.Sprintf("drone-%s-%d", cluster, i),
			Lat:       48.0 + rand.Float64(),
			Lon:       16.0 + rand.Float64(),
			Alt:       100 + rand.Float64()*20,
			Battery:   90 + rand.Float64()*10,
			Status:    "ok",
			TS:        time.Now().Add(time.Duration(i) * time.Minute),
		}
		if _, err := db.Exec(`insert into telemetry values (?,?,?,?,?,?,?,?)`,
			tel.ClusterID, tel.DroneID, tel.Lat, tel.Lon, tel.Alt, tel.Battery, tel.Status, tel.TS.Format(time.RFC3339)); err != nil {
			return err
		}
	}
	return nil
}

func ensureCommandTable(path string) error {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`create table if not exists telemetry (
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
	return err
}

func sync(srcPath, dstPath, mission string) error {
	src, err := sql.Open("sqlite3", srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := sql.Open("duckdb", dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	rows, err := src.Query(`select cluster_id, drone_id, lat, lon, alt, battery, status, ts from telemetry`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var t Telemetry
		var ts string
		if err := rows.Scan(&t.ClusterID, &t.DroneID, &t.Lat, &t.Lon, &t.Alt, &t.Battery, &t.Status, &ts); err != nil {
			return err
		}
		t.TS, _ = time.Parse(time.RFC3339, ts)
		if _, err := dst.Exec(`insert into telemetry values (?,?,?,?,?,?,?,?,?)`,
			t.ClusterID, t.DroneID, t.Lat, t.Lon, t.Alt, t.Battery, t.Status, t.TS.Format(time.RFC3339), mission); err != nil {
			return err
		}
	}
	return rows.Err()
}

func countRows(path string) (int, error) {
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
