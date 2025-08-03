package main

import (
	"context"
	"io"
	"log/slog"

	"sling-sync-wrapper/internal/logging"
)

func testContext() context.Context {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return logging.NewContext(context.Background(), logger)
}
