package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunRequiresReduceSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(nil, &stdout, &stderr)

	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "reduce") {
		t.Fatalf("run() stderr = %q, want message containing %q", stderr.String(), "reduce")
	}
}

func TestRunReduceRequiresCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"reduce", "input.json"},
		&stdout,
		&stderr,
	)

	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "command") {
		t.Fatalf("run() stderr = %q, want message containing %q", stderr.String(), "command")
	}
}

func TestRunReduceRejectsEmptyCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"reduce", "input.json", "--command"},
		&stdout,
		&stderr,
	)

	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "command") {
		t.Fatalf("run() stderr = %q, want message containing %q", stderr.String(), "command")
	}
}

func TestRunReduceRequiresFailureMatcher(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{
			"reduce",
			"input.json",
			"--command", "cat {input}",
		},
		&stdout,
		&stderr,
	)

	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "--exit-code") {
		t.Fatalf(
			"run() stderr = %q, want message containing %q",
			stderr.String(),
			"--exit-code",
		)
	}
}

func TestRunReduceRejectsInvalidExitCode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{
			"reduce",
			"input.json",
			"--command", "cat {input}",
			"--exit-code", "not-a-number",
		},
		&stdout,
		&stderr,
	)

	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "--exit-code") {
		t.Fatalf(
			"run() stderr = %q, want message containing %q",
			stderr.String(),
			"--exit-code",
		)
	}
}

func TestRunReduceRejectsMissingInputFile(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	inputPath := filepath.Join(t.TempDir(), "missing.json")

	exitCode := run(
		[]string{
			"reduce",
			inputPath,
			"--command", "cat {input}",
			"--stderr-contains", "DiscountValueError",
		},
		&stdout,
		&stderr,
	)

	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "missing.json") {
		t.Fatalf(
			"run() stderr = %q, want message containing %q",
			stderr.String(),
			"missing.json",
		)
	}
}

func TestRunReduceRejectsInputThatDoesNotReproduceFailure(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	inputPath := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(inputPath, []byte(`{"discount":-1}`), 0o600); err != nil {
		t.Fatal(err)
	}

	exitCode := run(
		[]string{
			"reduce",
			inputPath,
			"--command", `printf "DifferentError" >&2; test -f {input}; exit 7`,
			"--exit-code", "7",
			"--stderr-contains", "DiscountValueError",
		},
		&stdout,
		&stderr,
	)

	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), "does not reproduce") {
		t.Fatalf(
			"run() stderr = %q, want message containing %q",
			stderr.String(),
			"does not reproduce",
		)
	}
}

func TestRunReduceAcceptsReproducingInput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	inputPath := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(inputPath, []byte(`{"discount":-1}`), 0o600); err != nil {
		t.Fatal(err)
	}

	exitCode := run(
		[]string{
			"reduce",
			inputPath,
			"--command", `printf "DiscountValueError" >&2; test -f {input}; exit 7`,
			"--exit-code", "7",
			"--stderr-contains", "DiscountValueError",
		},
		&stdout,
		&stderr,
	)

	if exitCode != 0 {
		t.Fatalf(
			"run() exit code = %d, want 0; stderr = %q",
			exitCode,
			stderr.String(),
		)
	}
}

func TestRunReduceHonorsTimeout(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	inputPath := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(inputPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	exitCode := run(
		[]string{
			"reduce",
			inputPath,
			"--command", `test -f {input}; sleep 1; printf "DiscountValueError" >&2; exit 7`,
			"--exit-code", "7",
			"--stderr-contains", "DiscountValueError",
			"--timeout", "50ms",
		},
		&stdout,
		&stderr,
	)

	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("run() elapsed = %v, want prompt timeout", elapsed)
	}
	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want timeout failure")
	}
}

func TestRunReduceHonorsConfirmRuns(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.json")
	counterPath := filepath.Join(dir, "count")
	if err := os.WriteFile(inputPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	quotedCounterPath := "'" + strings.ReplaceAll(counterPath, "'", `'\''`) + "'"
	command := `test -f {input}; ` +
		`count=$(cat ` + quotedCounterPath + ` 2>/dev/null || echo 0); ` +
		`count=$((count + 1)); printf %s "$count" > ` + quotedCounterPath + `; ` +
		`if [ "$count" -eq 1 ]; then printf "DiscountValueError" >&2; ` +
		`else printf "DifferentError" >&2; fi; exit 7`

	exitCode := run(
		[]string{
			"reduce",
			inputPath,
			"--command", command,
			"--exit-code", "7",
			"--stderr-contains", "DiscountValueError",
			"--confirm-runs", "2",
		},
		&stdout,
		&stderr,
	)

	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want flaky-failure rejection")
	}
}

func TestRunReduceRejectsUnknownOption(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	inputPath := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(inputPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	exitCode := run(
		[]string{
			"reduce",
			inputPath,
			"--command", `test -f {input}; printf "DiscountValueError" >&2; exit 7`,
			"--exit-code", "7",
			"--unknown-option", "value",
		},
		&stdout,
		&stderr,
	)

	if exitCode == 0 {
		t.Fatal("run() exit code = 0, want unknown-option rejection")
	}
	if !strings.Contains(stderr.String(), "unknown-option") {
		t.Fatalf(
			"run() stderr = %q, want message containing %q",
			stderr.String(),
			"unknown-option",
		)
	}
}

func TestRunReduceAppliesTimeoutPerConfirmationRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	inputPath := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(inputPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	exitCode := run(
		[]string{
			"reduce",
			inputPath,
			"--command", `test -f {input}; sleep 0.15; printf "DiscountValueError" >&2; exit 7`,
			"--exit-code", "7",
			"--stderr-contains", "DiscountValueError",
			"--timeout", "250ms",
			"--confirm-runs", "2",
		},
		&stdout,
		&stderr,
	)

	if exitCode != 0 {
		t.Fatalf(
			"run() exit code = %d, want 0; stderr = %q",
			exitCode,
			stderr.String(),
		)
	}
}
