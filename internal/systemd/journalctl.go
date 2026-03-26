package systemd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// GetRecentLogs retrieves recent logs for a systemd service unit
func GetRecentLogs(unitName string, lines int) ([]string, error) {
	cmd := exec.Command("journalctl", getScopeFlag(), "-u", unitName,
		"-n", fmt.Sprintf("%d", lines),
		"--no-pager",
		"-o", "short-iso")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("journalctl failed: %w", err)
	}

	var logLines []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		logLines = append(logLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse journalctl output: %w", err)
	}

	return logLines, nil
}

// FollowLogs streams logs for a systemd service unit
// It returns a channel that receives log lines and a cleanup function
func FollowLogs(ctx context.Context, unitName string) (<-chan string, func() error, error) {
	cmd := exec.CommandContext(ctx, "journalctl", getScopeFlag(), "-u", unitName,
		"-f",
		"--no-pager",
		"-o", "short-iso")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start journalctl: %w", err)
	}

	logChan := make(chan string, 100)
	scanner := bufio.NewScanner(stdout)

	go func() {
		defer close(logChan)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case logChan <- scanner.Text():
			}
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			select {
			case <-ctx.Done():
			case logChan <- fmt.Sprintf("Error reading logs: %v", err):
			}
		}
	}()

	cleanup := func() error {
		// Cleanup must never block the UI thread. Try to kill and wait briefly.
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case err := <-done:
			return err
		case <-time.After(500 * time.Millisecond):
			// Give up waiting; the context-based kill should still terminate the process soon.
			return nil
		}
	}

	return logChan, cleanup, nil
}
