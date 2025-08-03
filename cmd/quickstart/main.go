// Program quickstart creates sample mission and command databases and syncs
// them using direct SQL statements. It runs entirely standalone and does not
// invoke the Sling CLI or the sling-sync-wrapper. For an example using the
// wrapper, build and run the code in `cmd/wrapper` instead.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"sling-sync-wrapper/internal/logging"
	"sling-sync-wrapper/internal/sampledb"
)

func main() {
	logger := logging.New()

	dir := "quickstart"
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Error("create quickstart dir", "err", err)
		os.Exit(1)
	}

	mission1 := filepath.Join(dir, "mission1.db")
	mission2 := filepath.Join(dir, "mission2.db")
	command := filepath.Join(dir, "command.db")

	if err := sampledb.CreateMissionDB(mission1, "mission1", 10); err != nil {
		logger.Error("setup mission1", "err", err)
		os.Exit(1)
	}
	if err := sampledb.CreateMissionDB(mission2, "mission2", 10); err != nil {
		logger.Error("setup mission2", "err", err)
		os.Exit(1)
	}
	if err := sampledb.EnsureCommandTable(command, true); err != nil {
		logger.Error("setup command db", "err", err)
		os.Exit(1)
	}

	if _, err := sampledb.Sync(mission1, command, "mission1", true); err != nil {
		logger.Error("sync mission1", "err", err)
		os.Exit(1)
	}
	if _, err := sampledb.Sync(mission2, command, "mission2", true); err != nil {
		logger.Error("sync mission2", "err", err)
		os.Exit(1)
	}

	cnt, err := sampledb.CountRows(command)
	if err != nil {
		logger.Error("count rows", "err", err)
		os.Exit(1)
	}
	fmt.Printf("Quickstart complete! %d rows synced into %s\n", cnt, command)
}
