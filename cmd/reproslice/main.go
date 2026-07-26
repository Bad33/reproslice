package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	outputPath := flags.String("output", "", "minimized JSON output path")

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

	if *outputPath == "" {
		*outputPath = defaultOutputPath(inputPath)
	}

	sameFile, err := pathsReferToSameFile(inputPath, *outputPath)
	if err != nil {
		fmt.Fprintf(stderr, "validate output path %q: %v\n", *outputPath, err)
		return 1
	}
	if sameFile {
		fmt.Fprintln(stderr, "--output must not overwrite the input file")
		return 2
	}

	original, err := reproslice.LoadJSON(inputPath)
	if err != nil {
		fmt.Fprintf(stderr, "load input %q: %v\n", inputPath, err)
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

	reduced, err := reproslice.Reduce(
		context.Background(),
		original,
		*command,
		spec,
	)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := writeJSONFile(*outputPath, reduced); err != nil {
		fmt.Fprintf(stderr, "write output %q: %v\n", *outputPath, err)
		return 1
	}

	return 0
}

func pathsReferToSameFile(firstPath, secondPath string) (bool, error) {
	firstAbsolute, err := filepath.Abs(firstPath)
	if err != nil {
		return false, err
	}

	secondAbsolute, err := filepath.Abs(secondPath)
	if err != nil {
		return false, err
	}

	if firstAbsolute == secondAbsolute {
		return true, nil
	}

	firstInfo, err := os.Stat(firstAbsolute)
	if err != nil {
		return false, err
	}

	secondInfo, err := os.Stat(secondAbsolute)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return os.SameFile(firstInfo, secondInfo), nil
}

func defaultOutputPath(inputPath string) string {
	extension := filepath.Ext(inputPath)
	base := inputPath[:len(inputPath)-len(extension)]
	return base + ".min.json"
}

func writeJSONFile(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}

	tempPath := file.Name()
	defer os.Remove(tempPath)

	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}
