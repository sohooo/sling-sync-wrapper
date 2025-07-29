package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SlingLogLine represents a single JSON log entry from the Sling CLI.
type SlingLogLine struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Rows    int    `json:"rows,omitempty"`
	Error   string `json:"error,omitempty"`
}

func runSlingOnce(ctx context.Context, pipeline, stateLocation, jobID string, span trace.Span) (int, error) {
	cmd := exec.CommandContext(ctx, "sling", "sync", "--config", pipeline, "--log-format", "json")
	cmd.Env = append(os.Environ(), fmt.Sprintf("SLING_STATE=%s", stateLocation))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(stdout)
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
			span.AddEvent(line)
		}
	}

	if err := cmd.Wait(); err != nil {
		return rowsSynced, err
	}
	return rowsSynced, nil
}

func statusFromErr(err error) string {
	if err != nil {
		return "failed"
	}
	return "success"
}
