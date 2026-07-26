package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/Bad33/reproslice"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, _ io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "reduce" {
		fmt.Fprintln(stderr, "usage: reproslice reduce <input.json> [options]")
		return 2
	}

	if len(args) < 2 {
		fmt.Fprintln(stderr, "reduce requires an input JSON file")
		return 2
	}

	inputPath := args[1]

	flags := flag.NewFlagSet("reduce", flag.ContinueOnError)
	flags.SetOutput(stderr)

	command := flags.String("command", "", "command containing {input}")
	exitCodeValue := flags.String("exit-code", "", "expected exit code")
	stdoutContains := flags.String("stdout-contains", "", "required stdout substring")
	stderrContains := flags.String("stderr-contains", "", "required stderr substring")
	timeoutValue := flags.String("timeout", "", "per-command timeout")
	confirmRunsValue := flags.String("confirm-runs", "1", "required confirmation runs")

	if err := flags.Parse(args[2:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected argument: %s\n", flags.Arg(0))
		return 2
	}
	if *command == "" {
		fmt.Fprintln(stderr, "reduce requires a non-empty --command")
		return 2
	}

	spec := reproslice.FailureSpec{
		StdoutContains: *stdoutContains,
		StderrContains: *stderrContains,
	}

	if *exitCodeValue != "" {
		exitCode, err := strconv.Atoi(*exitCodeValue)
		if err != nil {
			fmt.Fprintln(stderr, "--exit-code must be an integer")
			return 2
		}
		spec.ExitCode = &exitCode
	}

	if spec.ExitCode == nil && spec.StdoutContains == "" && spec.StderrContains == "" {
		fmt.Fprintln(
			stderr,
			"reduce requires at least one of --exit-code, --stdout-contains, or --stderr-contains",
		)
		return 2
	}

	confirmRuns, err := strconv.Atoi(*confirmRunsValue)
	if err != nil || confirmRuns < 1 {
		fmt.Fprintln(stderr, "--confirm-runs must be a positive integer")
		return 2
	}
	spec.ConfirmRuns = confirmRuns

	if _, err := reproslice.LoadJSON(inputPath); err != nil {
		fmt.Fprintf(stderr, "load input %q: %v\n", inputPath, err)
		return 1
	}

	candidate, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(stderr, "read input %q: %v\n", inputPath, err)
		return 1
	}

	if *timeoutValue != "" {
		timeout, err := time.ParseDuration(*timeoutValue)
		if err != nil || timeout <= 0 {
			fmt.Fprintln(stderr, "--timeout must be a positive duration")
			return 2
		}
		spec.Timeout = timeout
	}

	if err := reproslice.VerifyOriginal(
		context.Background(),
		*command,
		candidate,
		spec,
	); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	return 0
}
