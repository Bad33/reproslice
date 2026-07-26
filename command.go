package reproslice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type commandResult struct {
	exitCode int
	stdout   []byte
	stderr   []byte
}

func runCommand(ctx context.Context, command string, candidate []byte) (commandResult, error) {
	if !strings.Contains(command, "{input}") {
		return commandResult{}, fmt.Errorf("command must contain {input} placeholder")
	}

	file, err := os.CreateTemp("", "reproslice-*.json")
	if err != nil {
		return commandResult{}, err
	}
	path := file.Name()
	defer os.Remove(path)

	if _, err := file.Write(candidate); err != nil {
		file.Close()
		return commandResult{}, err
	}
	if err := file.Close(); err != nil {
		return commandResult{}, err
	}

	command = strings.ReplaceAll(command, "{input}", shellQuote(path))

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}

	err = cmd.Run()

	result := commandResult{
		stdout: stdout.Bytes(),
		stderr: stderr.Bytes(),
	}

	if ctx.Err() != nil {
		return commandResult{}, ctx.Err()
	}
	if err == nil {
		return result, nil
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		exitCode := exitError.ExitCode()
		if exitCode == 127 {
			return commandResult{}, fmt.Errorf("command failed to start: %s", strings.TrimSpace(string(result.stderr)))
		}
		result.exitCode = exitCode
		return result, nil
	}

	return commandResult{}, err
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
