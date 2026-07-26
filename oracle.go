package reproslice

import (
	"bytes"
	"context"
	"fmt"
	"time"
)

type failureSpec struct {
	exitCode       *int
	stdoutContains string
	stderrContains string
}

func validateFailureSpec(spec failureSpec) error {
	if spec.exitCode == nil && spec.stdoutContains == "" && spec.stderrContains == "" {
		return fmt.Errorf("at least one failure matcher is required")
	}

	return nil
}

func matchesFailure(result commandResult, spec failureSpec) bool {
	if spec.exitCode != nil && result.exitCode != *spec.exitCode {
		return false
	}
	if spec.stdoutContains != "" && !bytes.Contains(result.stdout, []byte(spec.stdoutContains)) {
		return false
	}
	if spec.stderrContains != "" && !bytes.Contains(result.stderr, []byte(spec.stderrContains)) {
		return false
	}

	return true
}

func verifyOriginal(
	ctx context.Context,
	command string,
	candidate []byte,
	spec failureSpec,
) error {
	if err := validateFailureSpec(spec); err != nil {
		return err
	}

	result, err := runCommand(ctx, command, candidate)
	if err != nil {
		return fmt.Errorf("run original payload: %w", err)
	}
	if !matchesFailure(result, spec) {
		return fmt.Errorf("original payload does not reproduce the expected failure")
	}

	return nil
}

type FailureSpec struct {
	ExitCode       *int
	StdoutContains string
	StderrContains string
	ConfirmRuns    int
	Timeout        time.Duration
}

func normalizeFailureSpec(spec FailureSpec) (failureSpec, int, error) {
	confirmRuns := spec.ConfirmRuns
	if confirmRuns == 0 {
		confirmRuns = 1
	}
	if confirmRuns < 1 {
		return failureSpec{}, 0, fmt.Errorf("confirmation runs must be at least 1")
	}
	if spec.Timeout < 0 {
		return failureSpec{}, 0, fmt.Errorf("timeout must not be negative")
	}

	internalSpec := failureSpec{
		exitCode:       spec.ExitCode,
		stdoutContains: spec.StdoutContains,
		stderrContains: spec.StderrContains,
	}
	if err := validateFailureSpec(internalSpec); err != nil {
		return failureSpec{}, 0, err
	}

	return internalSpec, confirmRuns, nil
}

func VerifyOriginal(
	ctx context.Context,
	command string,
	candidate []byte,
	spec FailureSpec,
) error {
	internalSpec, confirmRuns, err := normalizeFailureSpec(spec)
	if err != nil {
		return err
	}

	for run := 1; run <= confirmRuns; run++ {
		runCtx := ctx
		cancel := func() {}
		if spec.Timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, spec.Timeout)
		}

		err := verifyOriginal(runCtx, command, candidate, internalSpec)
		cancel()
		if err != nil {
			return fmt.Errorf("confirmation run %d of %d: %w", run, confirmRuns, err)
		}
	}

	return nil
}
