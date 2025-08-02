# Sling Sync Wrapper (with OpenTelemetry)

## Overview

This project wraps the Sling CLI to:

- Run data sync jobs from mission clusters to a central command cluster.
- Emit OpenTelemetry traces and logs for rich observability.
- Support multiple pipelines, retry logic, backfill mode, and noop (dry-run) mode.
- Integrate easily into Kubernetes as a CronJob.

The collected telemetry is stored in GreptimeDB and visualized via Grafana.

## Features

**OpenTelemetry Tracing & Logging**; Each sync run is traced:

    - sync_job_id, rows_synced, duration_seconds, and status.
    - Logs captured as span events.
    - Retry Logic; Retries failed syncs using exponential backoff (SYNC_MAX_RETRIES, SYNC_BACKOFF_BASE).

**Multi-Pipeline Support**

- Mount multiple pipeline YAML files and run them sequentially.

**Modes**

- noop: validate pipelines and environment, but don’t execute sync.
- backfill: clear state and perform a full historical sync.

**Drill-Down Links in Grafana**

- Jump from traces → logs and logs → traces for rapid troubleshooting.

## Architecture

```
┌────────────────────┐        ┌────────────────────────────┐
│   Sling Wrapper    │        │  OpenTelemetry Collector   │
│  (CronJob, Go)     │        │ (Deployment, OTLP->GreptimeDB)│
└─────────┬──────────┘        └─────────────┬──────────────┘
          │ OpenTelemetry (Traces & Logs)   │
          v                                 v
    ┌──────────────┐                 ┌───────────────┐
    │  GreptimeDB  │ <── (Optional) ─│ Sling State DB│
    └──────────────┘                 └───────────────┘
          │
          v
     ┌─────────┐
     │ Grafana │
     └─────────┘
```

## QUICKSTART

The quickstart synchronizes two sample SQLite mission databases into a single
DuckDB command database. It generates realistic drone telemetry, writes simple
Sling pipeline files and performs a sync just like the integration test
`sqlite_duckdb_test.go`.

Run everything with one command:

```bash
make quickstart
```

After the command finishes you will find the databases and pipelines under the
`quickstart/` directory and a `command.db` file populated with telemetry from
both missions. Inspect the YAML files in `quickstart/pipelines/` to see how a
minimal `SLING_CONFIG` looks. From here you can adjust the pipelines or point
the wrapper at your own databases.

## Quickstart (Local)

### Prerequisites

- Go 1.22+
- Sling CLI installed
- GreptimeDB and Grafana (optional for observability)

### Build

```bash
go build -o sling-sync-wrapper ./cmd/wrapper
```

### Run Example

```bash
MISSION_CLUSTER_ID=mission-01 \
SLING_CONFIG=./pipeline.yaml \
SLING_STATE=file://./sling_state.json \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
./sling-sync-wrapper
```

The wrapper automatically generates a unique `SYNC_JOB_ID` for each run.

### Modes

- noop: dry-run (no sync)
- backfill: reset state and re-sync all data

```bash
# noop
SYNC_MODE=noop ./sling-sync-wrapper

# backfil
SYNC_MODE=backfill ./sling-sync-wrapper
```

## Environment Variables

The wrapper is configured using the following environment variables:

| Variable | Default | Required? | Description |
|----------|---------|-----------|-------------|
| `MISSION_CLUSTER_ID` | `unknown-cluster` | Yes | Source cluster identifier used in telemetry. |
| `SLING_CONFIG` | – | Yes* | Path to a single pipeline file. Required if `PIPELINE_DIR` is not set. |
| `PIPELINE_DIR` | `/etc/sling/pipelines` | Yes* | Directory containing one or more pipeline files. Required if `SLING_CONFIG` is not set. |
| `SLING_STATE` | `file://./sling_state.json` | No | Path or URL where sync state is stored. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `otel-collector:4317` | No | OpenTelemetry Collector endpoint for traces and logs. |
| `SYNC_MODE` | `normal` | No | Sync mode: `normal` (incremental), `noop`, or `backfill`. |
| `SYNC_MAX_RETRIES` | `3` | No | Number of times to retry a failed pipeline run. |
| `SYNC_BACKOFF_BASE` | `5s` | No | Base duration for exponential backoff between retries. |
| `SLING_BIN` | `sling` | No | Path to the Sling CLI binary. |

`*` Either `SLING_CONFIG` or `PIPELINE_DIR` must be set.

## Kubernetes Deployment

### Apply manifests

```bash
kubectl apply -f deploy/deployment.yaml
```

This will deploy:

- OpenTelemetry Collector (Deployment + Service).
- Sling Sync Job (CronJob) running every 5 minutes.
- Pipeline ConfigMap (sling-pipelines) for one or more pipeline YAMLs.

Configure the CronJob using the environment variables described in the [Environment Variables](#environment-variables) section.

## Grafana Dashboard

A pre-built Grafana dashboard is provided:

- Panels:
  - Job counts and status breakdown.
  - Sync duration and row trends.
  - Recent jobs and logs with drill-down links.
- Links:
  - Trace → Logs.
  - Logs → Trace Explorer.
          
### Import Dashboard

- Open Grafana → Dashboards → Import.
- Paste the provided JSON.
- Set:
  - `YOUR_TRACE_DATASOURCE_UID` → GreptimeDB traces.
  - `YOUR_LOGS_DATASOURCE_UID` → GreptimeDB logs.

## Development

### Maintenance

Common development tasks are available via the Makefile:

```bash
make fmt   # format Go code
make vet   # run go vet
make tidy  # tidy module dependencies
make test  # run unit tests
```

Run locally with multiple pipelines

```bash
PIPELINE_DIR=./pipelines \
MISSION_CLUSTER_ID=local \
./sling-sync-wrapper
```

### Retry Testing

```bash
SYNC_MAX_RETRIES=5 SYNC_BACKOFF_BASE=2s ./sling-sync-wrapper
```

## Observability

- Traces:
  - One trace per sync job.
  - Span attributes include job metadata.
- Logs:
  - Each Sling log message attached as a span event.
  - Metrics (optional future work):
  - Could export Prometheus metrics for job success/failure counters.

## Next Steps / Enhancements

- Add Prometheus metrics for sync jobs.
- Add schema/state validation pre-run.
- Add alerting rules for repeated failures.
