package main

import (
	"os"
	"path/filepath"
	"testing"

	"sling-sync-wrapper/internal/config"
)

func TestResetStateRemovesFile(t *testing.T) {
	var removed string
	removeAllFunc = func(path string) error {
		removed = path
		return nil
	}
	defer func() { removeAllFunc = os.RemoveAll }()

	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")
	cfg := config.Config{StateLocation: "file://" + stateFile}
	if err := resetState(testContext(), cfg); err != nil {
		t.Fatalf("resetState returned error: %v", err)
	}
	if removed != stateFile {
		t.Fatalf("removeAll called with %q, want %q", removed, stateFile)
	}
}

func TestResetStateSkipsNonFileScheme(t *testing.T) {
	var called bool
	removeAllFunc = func(path string) error {
		called = true
		return nil
	}
	defer func() { removeAllFunc = os.RemoveAll }()

	cfg := config.Config{StateLocation: "s3://bucket/state.json"}
	if err := resetState(testContext(), cfg); err != nil {
		t.Fatalf("resetState returned error: %v", err)
	}
	if called {
		t.Fatalf("removeAll should not be called for non-file scheme")
	}
}
