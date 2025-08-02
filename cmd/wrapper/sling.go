package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var execCommandContext = exec.CommandContext

// SlingLogLine represents a single JSON log entry from the Sling CLI.
type SlingLogLine struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Rows    int    `json:"rows,omitempty"`
	Error   string `json:"error,omitempty"`
}

const (
	maxScanTokenSize = 1024 * 1024 // 1 MiB
	slingCLITimeout  = 30 * time.Minute
)

func runSlingOnce(ctx context.Context, slingBin, pipeline, stateLocation, jobID string, span trace.Span) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, slingCLITimeout)
	defer cancel()

	cmd := execCommandContext(ctx, slingBin, "sync", "--config", pipeline, "--log-format", "json")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SLING_STATE=%s", stateLocation),
		fmt.Sprintf("SYNC_JOB_ID=%s", jobID),
		fmt.Sprintf("SLING_CONFIG=%s", pipeline),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)
	rowsSynced := 0
	for scanner.Scan() {
		line := scanner.Text()
		var logEntry SlingLogLine
		if err := json.Unmarshal([]byte(line), &logEntry); err == nil {
			span.AddEvent(logEntry.Message,
				trace.WithAttributes(attribute.String("log.level", logEntry.Level)))
			if logEntry.Rows > 0 {
				rowsSynced += logEntry.Rows
			}
			if logEntry.Error != "" {
				span.RecordError(fmt.Errorf("%s", logEntry.Error))
			}
		} else {
			log.Printf("failed to parse Sling log line: %v", err)
			span.RecordError(err)
			span.AddEvent("invalid JSON log line",
				trace.WithAttributes(attribute.String("line", line)))
		}
	}

	if err := scanner.Err(); err != nil {
		cmd.Wait() // ensure process resources are released
		return rowsSynced, err
	}

	if err := cmd.Wait(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return rowsSynced, ctxErr
		}
		return rowsSynced, err
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return rowsSynced, ctxErr
	}
	return rowsSynced, nil
}

func statusFromErr(err error) string {
	if err != nil {
		return "failed"
	}
	return "success"
}
