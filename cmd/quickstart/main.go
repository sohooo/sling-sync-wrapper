// Program quickstart creates sample mission and command databases and syncs
// them using direct SQL statements. It runs entirely standalone and does not
// invoke the Sling CLI or the sling-sync-wrapper. For an example using the
// wrapper, build and run the code in `cmd/wrapper` instead.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"sling-sync-wrapper/internal/sampledb"
)

func main() {
	dir := "quickstart"
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal(err)
	}

	mission1 := filepath.Join(dir, "mission1.db")
	mission2 := filepath.Join(dir, "mission2.db")
	command := filepath.Join(dir, "command.db")

	if err := sampledb.CreateMissionDB(mission1, "mission1", 10); err != nil {
		log.Fatalf("setup mission1: %v", err)
	}
	if err := sampledb.CreateMissionDB(mission2, "mission2", 10); err != nil {
		log.Fatalf("setup mission2: %v", err)
	}
	if err := sampledb.EnsureCommandTable(command, true); err != nil {
		log.Fatalf("setup command db: %v", err)
	}

	if _, err := sampledb.Sync(mission1, command, "mission1", true); err != nil {
		log.Fatal(err)
	}
	if _, err := sampledb.Sync(mission2, command, "mission2", true); err != nil {
		log.Fatal(err)
	}

	cnt, err := sampledb.CountRows(command)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Quickstart complete! %d rows synced into %s\n", cnt, command)
}
