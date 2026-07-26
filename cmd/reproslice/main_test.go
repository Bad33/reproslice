package main

import (
	"bytes"
	"encoding/json"
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

func TestRunReduceWritesMinimizedOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.json")
	outputPath := filepath.Join(dir, "input.min.json")

	original := []byte(`{"required":"keep","noise":"drop"}`)
	if err := os.WriteFile(inputPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	exitCode := run(
		[]string{
			"reduce",
			inputPath,
			"--command", `if grep -q '"required":"keep"' {input}; then printf "TargetError" >&2; exit 7; fi`,
			"--exit-code", "7",
			"--stderr-contains", "TargetError",
			"--output", outputPath,
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

	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if got["required"] != "keep" {
		t.Fatalf("output required = %#v, want %q", got["required"], "keep")
	}
	if _, exists := got["noise"]; exists {
		t.Fatalf("output still contains removable field: %s", output)
	}

	after, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatalf("input changed from %q to %q", original, after)
	}
}

func TestRunReduceUsesDefaultOutputPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.json")
	outputPath := filepath.Join(dir, "input.min.json")

	if err := os.WriteFile(
		inputPath,
		[]byte(`{"required":"keep","noise":"drop"}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	exitCode := run(
		[]string{
			"reduce",
			inputPath,
			"--command", `if grep -q '"required":"keep"' {input}; then printf "TargetError" >&2; exit 7; fi`,
			"--exit-code", "7",
			"--stderr-contains", "TargetError",
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

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("default output %q: %v", outputPath, err)
	}
}

func TestRunReduceWritesTheTestedSerialization(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.json")
	outputPath := filepath.Join(dir, "output.json")

	if err := os.WriteFile(
		inputPath,
		[]byte(`{"required":"keep","noise":"drop"}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	exitCode := run(
		[]string{
			"reduce",
			inputPath,
			"--command",
			`content=$(cat {input}); case "$content" in '{"noise":"drop","required":"keep"}'|'{"required":"keep"}') exit 7;; *) exit 0;; esac`,
			"--exit-code", "7",
			"--output", outputPath,
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

	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	const want = `{"required":"keep"}`
	if string(output) != want {
		t.Fatalf("output = %q, want exact tested bytes %q", output, want)
	}
}

func TestRunReduceRejectsOutputPathMatchingInput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.json")
	original := []byte(`{"required":"keep","noise":"drop"}`)

	if err := os.WriteFile(inputPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	exitCode := run(
		[]string{
			"reduce",
			inputPath,
			"--command", `if grep -q '"required":"keep"' {input}; then exit 7; fi`,
			"--exit-code", "7",
			"--output", inputPath,
		},
		&stdout,
		&stderr,
	)
	if exitCode != 2 {
		t.Fatalf(
			"run() exit code = %d, want 2; stderr = %q",
			exitCode,
			stderr.String(),
		)
	}

	after, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatalf("input changed from %q to %q", original, after)
	}
}
