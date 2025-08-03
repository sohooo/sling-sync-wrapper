package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"sling-sync-wrapper/internal/config"
	"sling-sync-wrapper/internal/logging"
)

func resetState(ctx context.Context, cfg config.Config) error {
	logger := logging.FromContext(ctx)
	logger.Info("resetting sync state", "mode", "backfill", "state_location", cfg.StateLocation)
	u, err := url.Parse(cfg.StateLocation)
	if err != nil {
		logger.Error("invalid state location", "err", err)
		return fmt.Errorf("parse state location: %w", err)
	}
	if u.Scheme != "" && u.Scheme != "file" {
		logger.Error("state location scheme not supported for backfill", "scheme", u.Scheme)
		return nil
	}
	p := u.Path
	if p == "" {
		p = u.Opaque
	}
	p = filepath.Clean(p)
	if p == "." || p == string(os.PathSeparator) {
		logger.Error("state location path is unsafe; skipping reset", "path", p)
		return nil
	}
	if err := removeAllFunc(p); err != nil {
		logger.Error("failed to reset state", "err", err)
		return fmt.Errorf("remove state path %s: %w", p, err)
	}
	return nil
}
