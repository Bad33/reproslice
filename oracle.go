package reproslice

import (
	"bytes"
	"context"
	"fmt"
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
