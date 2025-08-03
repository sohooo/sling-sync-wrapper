package main

import (
	"testing"
)

// TestNewRootCmdEnv verifies that environment variables are used as defaults for CLI flags.
func TestNewRootCmdEnv(t *testing.T) {
	t.Setenv("MISSION_CLUSTER_ID", "env-mission")
	t.Setenv("SLING_CONFIG", "env-pipeline.yaml")
	t.Setenv("SLING_STATE", "env-state.json")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-env:4317")
	t.Setenv("SYNC_MAX_RETRIES", "7")
	t.Setenv("SYNC_BACKOFF_BASE", "3s")
	t.Setenv("SLING_BIN", "/env/sling")
	t.Setenv("SLING_TIMEOUT", "45s")

	cmd := newRootCmd()

	tests := []struct {
		flag string
		want string
	}{
		{"mission-cluster-id", "env-mission"},
		{"config", "env-pipeline.yaml"},
		{"state", "env-state.json"},
		{"otel-endpoint", "otel-env:4317"},
		{"max-retries", "7"},
		{"backoff-base", "3s"},
		{"sling-binary", "/env/sling"},
		{"sling-timeout", "45s"},
	}

	for _, tt := range tests {
		got := cmd.PersistentFlags().Lookup(tt.flag).Value.String()
		if got != tt.want {
			t.Errorf("flag %s = %s, want %s", tt.flag, got, tt.want)
		}
	}
}
