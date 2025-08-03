package main

import (
	"context"

	"github.com/spf13/cobra"

	"sling-sync-wrapper/internal/config"
)

// newRootCmd constructs the root command for the wrapper CLI.
func newRootCmd() *cobra.Command {
	cfg := config.FromEnv()

	cmd := &cobra.Command{
		Use:   "sling-sync-wrapper",
		Short: "Run Sling sync pipelines with tracing and retries",
		Long: `Sling Sync Wrapper orchestrates Sling pipeline executions with
telemetry, retry logic, and state management. Configuration can be
supplied via flags or environment variables.`,
		SilenceUsage: true,
	}

	cmd.PersistentFlags().StringVar(&cfg.MissionClusterID, "mission-cluster-id", cfg.MissionClusterID, "Source mission cluster identifier (env: MISSION_CLUSTER_ID)")
	cmd.PersistentFlags().StringVar(&cfg.PipelineFile, "config", cfg.PipelineFile, "Path to a single pipeline YAML file (env: SLING_CONFIG)")
	cmd.PersistentFlags().StringVar(&cfg.PipelineDir, "pipeline-dir", cfg.PipelineDir, "Directory containing pipeline YAML files (env: PIPELINE_DIR)")
	cmd.PersistentFlags().StringVar(&cfg.StateLocation, "state", cfg.StateLocation, "URI where sync state is stored (env: SLING_STATE)")
	cmd.PersistentFlags().StringVar(&cfg.OTELEndpoint, "otel-endpoint", cfg.OTELEndpoint, "OpenTelemetry collector endpoint (env: OTEL_EXPORTER_OTLP_ENDPOINT)")
	cmd.PersistentFlags().IntVar(&cfg.MaxRetries, "max-retries", cfg.MaxRetries, "Maximum retry attempts for failed syncs (env: SYNC_MAX_RETRIES)")
	cmd.PersistentFlags().DurationVar(&cfg.BackoffBase, "backoff-base", cfg.BackoffBase, "Base duration for exponential backoff (env: SYNC_BACKOFF_BASE)")
	cmd.PersistentFlags().StringVar(&cfg.SlingBinary, "sling-binary", cfg.SlingBinary, "Path to the Sling CLI binary (env: SLING_BIN)")
	cmd.PersistentFlags().DurationVar(&cfg.SlingTimeout, "sling-timeout", cfg.SlingTimeout, "Maximum duration for a single Sling run (env: SLING_TIMEOUT)")

	cmd.AddCommand(newRunCmd(cfg), newBackfillCmd(cfg), newNoopCmd(cfg))

	return cmd
}

func newRunCmd(cfg config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run configured pipelines",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.SyncMode = "normal"
			return run(context.Background(), cfg)
		},
	}
}

func newBackfillCmd(cfg config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "backfill",
		Short: "Reset sync state and exit",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.SyncMode = "backfill"
			return run(context.Background(), cfg)
		},
	}
}

func newNoopCmd(cfg config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "noop",
		Short: "Validate configuration without running pipelines",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.SyncMode = "noop"
			return run(context.Background(), cfg)
		},
	}
}

// Execute runs the CLI.
func Execute() error {
	return newRootCmd().Execute()
}
