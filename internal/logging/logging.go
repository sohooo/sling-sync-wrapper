package logging

import (
	"context"
	"log/slog"
	"os"
)

// New returns a JSON logger writing to stderr with source information.
func New() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{AddSource: true})
	return slog.New(handler)
}

type ctxKey struct{}

// NewContext returns a copy of ctx with logger attached.
func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// FromContext retrieves a logger from ctx or returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}
