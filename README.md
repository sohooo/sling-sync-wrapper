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
SYNC_JOB_ID=$(uuidgen) \
SLING_CONFIG=./pipeline.yaml \
SLING_STATE_PATH=./sling_state \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
./sling-sync-wrapper
```

### Modes

- noop: dry-run (no sync)
- backfill: reset state and re-sync all data

```bash
# noop
SYNC_MODE=noop ./sling-sync-wrapper

# backfil
SYNC_MODE=backfill ./sling-sync-wrapper
```

## Kubernetes Deployment

### Apply manifests

```bash
kubectl apply -f deploy/deployment.yaml
```

This will deploy:

- OpenTelemetry Collector (Deployment + Service).
- Sling Sync Job (CronJob) running every 5 minutes.
- Pipeline ConfigMap (sling-pipelines) for one or more pipeline YAMLs.

### Environment Variables

- `MISSION_CLUSTER_ID` (required) → source cluster identifier.
- `PIPELINE_DIR` → directory with multiple pipelines (default /etc/sling/pipelines).
- `SYNC_MODE` → "" (incremental), noop, backfill.
- `SLING_STATE_PATH` → path to sync state (default /var/lib/sling_state).
- `SYNC_MAX_RETRIES` → number of retries (default 3).
- `SYNC_BACKOFF_BASE` → backoff base (default 5s).
- `OTEL_EXPORTER_OTLP_ENDPOINT` → OTel Collector endpoint (default otel-collector:4317).
- `SLING_BIN` → path to the Sling CLI binary (default `sling`).

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
