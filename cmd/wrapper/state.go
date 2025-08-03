package main

import (
	"log"
	"net/url"
	"os"
	"path/filepath"

	"sling-sync-wrapper/internal/config"
)

func resetState(cfg config.Config) error {
	log.Printf("[BACKFILL] Resetting sync state at %s", cfg.StateLocation)
	u, err := url.Parse(cfg.StateLocation)
	if err != nil {
		log.Printf("Invalid state location: %v", err)
		return err
	}
	if u.Scheme != "" && u.Scheme != "file" {
		log.Printf("State location scheme %q is not supported for backfill", u.Scheme)
		return nil
	}
	p := u.Path
	if p == "" {
		p = u.Opaque
	}
	p = filepath.Clean(p)
	if p == "." || p == string(os.PathSeparator) {
		log.Printf("State location path %q is unsafe; skipping reset", p)
		return nil
	}
	if err := removeAllFunc(p); err != nil {
		log.Printf("Failed to reset state: %v", err)
		return err
	}
	return nil
}
