package reproslice

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunCommandRequiresInputPlaceholder(t *testing.T) {
	_, err := runCommand(t.Context(), "printf hello", nil)
	if err == nil {
		t.Fatal("runCommand() error = nil, want missing-placeholder error")
	}
	if !strings.Contains(err.Error(), "{input}") {
		t.Fatalf("runCommand() error = %q, want message containing {input}", err)
	}
}

func TestRunCommandCapturesStandardOutput(t *testing.T) {
	result, err := runCommand(t.Context(), "cat {input}", []byte(`{"name":"reproslice"}`))
	if err != nil {
		t.Fatalf("runCommand() error = %v", err)
	}
	if string(result.stdout) != `{"name":"reproslice"}` {
		t.Fatalf("runCommand() stdout = %q, want candidate JSON", result.stdout)
	}
}

func TestRunCommandCapturesNonZeroExit(t *testing.T) {
	result, err := runCommand(
		t.Context(),
		`printf "boom" >&2; test -f {input}; exit 7`,
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("runCommand() error = %v, want nil for completed command", err)
	}
	if result.exitCode != 7 {
		t.Fatalf("runCommand() exitCode = %d, want 7", result.exitCode)
	}
	if string(result.stderr) != "boom" {
		t.Fatalf("runCommand() stderr = %q, want %q", result.stderr, "boom")
	}
}

func TestRunCommandReturnsTimeoutError(t *testing.T) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	_, err := runCommand(ctx, `sleep 1; cat {input}`, []byte(`{}`))
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("runCommand() elapsed = %v, want prompt timeout", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("runCommand() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestRunCommandRejectsMissingExecutable(t *testing.T) {
	result, err := runCommand(
		t.Context(),
		`reproslice-command-does-not-exist {input}`,
		[]byte(`{}`),
	)
	if err == nil {
		t.Fatalf("runCommand() = %+v, nil; want command-start error", result)
	}
}

func TestRunCommandRemovesTemporaryFile(t *testing.T) {
	result, err := runCommand(t.Context(), `printf %s {input}`, []byte(`{}`))
	if err != nil {
		t.Fatalf("runCommand() error = %v", err)
	}

	path := string(result.stdout)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("temporary file %q still exists; os.Stat() error = %v", path, err)
	}
}
