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
)

var slingCLITimeout = 30 * time.Minute

// processLogLine parses a JSON line from the Sling CLI and updates the span.
// It returns the number of rows contained in the log or an error if the line
// could not be parsed.
func processLogLine(line string, span trace.Span) (int, error) {
	var logEntry SlingLogLine
	if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
		span.RecordError(err)
		span.AddEvent("invalid JSON log line",
			trace.WithAttributes(attribute.String("line", line)))
		return 0, fmt.Errorf("decode log line: %w", err)
	}

	span.AddEvent(logEntry.Message,
		trace.WithAttributes(attribute.String("log.level", logEntry.Level)))
	if logEntry.Error != "" {
		span.RecordError(fmt.Errorf("%s", logEntry.Error))
	}

	return logEntry.Rows, nil
}

// checkSlingErrors combines errors from the scanner, command wait, and context
// to produce a single error result.
func checkSlingErrors(ctx context.Context, cmd *exec.Cmd, scanErr error) error {
	if scanErr != nil {
		cmd.Wait() // ensure process resources are released
		return fmt.Errorf("scan sling output: %w", scanErr)
	}
	if err := cmd.Wait(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("command context: %w", ctxErr)
		}
		return fmt.Errorf("wait for sling: %w", err)
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("command context: %w", ctxErr)
	}
	return nil
}

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
		return 0, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start sling: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 0, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)
	rowsSynced := 0
	for scanner.Scan() {
		rows, err := processLogLine(scanner.Text(), span)
		if err != nil {
			log.Printf("failed to parse Sling log line: %v", err)
			continue
		}
		if rows > 0 {
			rowsSynced += rows
		}
	}

	if err := checkSlingErrors(ctx, cmd, scanner.Err()); err != nil {
		return rowsSynced, fmt.Errorf("execute sling: %w", err)
	}

	return rowsSynced, nil
}

func statusFromErr(err error) string {
	if err != nil {
		return "failed"
	}
	return "success"
}
