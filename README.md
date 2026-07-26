# ReproSlice

ReproSlice is a Go CLI and library that minimizes a failing JSON payload while preserving the same externally observed failure.

It removes or simplifies parts of the JSON, executes a user-supplied command for each candidate, and keeps only changes that still match the requested failure.

## Status

ReproSlice `v0.1.0` is released and available as a Go library and CLI.

The initial release is intentionally JSON-only and has zero runtime dependencies outside the Go standard library.

## Requirements

- Go 1.26 or newer
- A Unix-like operating system with `sh`

## Install

```sh
go install github.com/Bad33/reproslice/cmd/reproslice@latest
```

To build from a local checkout:

```sh
go build ./cmd/reproslice
```

## Usage

```text
reproslice reduce <input.json> [options]
```

Required:

- `--command`: a shell command containing the `{input}` placeholder
- At least one of:
  - `--exit-code`
  - `--stdout-contains`
  - `--stderr-contains`

Optional:

- `--confirm-runs`: consecutive matching runs required for each payload; default `1`
- `--timeout`: timeout applied separately to every command run
- `--output`: output path; default `<input-base>.min.json`

When multiple failure matchers are supplied, all must match.

## Example

Given `input.json`:

```json
{
  "required": "keep",
  "noise": "drop"
}
```

Run:

```sh
reproslice reduce input.json \
  --command "grep -q '\"required\":\"keep\"' {input} && { printf TargetError >&2; exit 7; }" \
  --exit-code 7 \
  --stderr-contains TargetError
```

The minimized payload is written to `input.min.json`:

```json
{"required":"keep"}
```

ReproSlice replaces every `{input}` occurrence with the safely quoted path of a temporary file containing the current candidate. Commands run through `sh -c`.

## Reduction behavior

ReproSlice currently attempts:

- Deterministic object-key removal
- Contiguous array chunk removal
- Individual array-element removal
- Recursive object and array reduction
- Fixed-point passes until no further change succeeds
- String simplification to `""`, the first Unicode character, or `null`
- Number simplification to `0`, `1`, `-1`, or `null`
- Boolean simplification to `false` or `null`

The original in-memory value and input file remain unchanged.

The CLI writes the exact compact JSON serialization tested by the failure command. It refuses output paths that refer directly or indirectly to the input file and writes output through a temporary file followed by rename.

Within one reduction, byte-identical serialized candidates reuse their cached oracle result instead of executing the command again.

## Failure verification

Before reduction starts, ReproSlice verifies that the original payload matches the requested failure.

`--confirm-runs` requires consecutive matches for both the original payload and every candidate. A fresh `--timeout` applies to each confirmation run.

Ordinary nonzero exit codes are command results that may be matched. A timeout, command-start failure, or other execution error stops the reduction.

## Library usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/Bad33/reproslice"
)

func main() {
	original, err := reproslice.LoadJSON("input.json")
	if err != nil {
		panic(err)
	}

	expectedExitCode := 7

	reduced, err := reproslice.Reduce(
		context.Background(),
		original,
		`processor {input}`,
		reproslice.FailureSpec{
			ExitCode:       &expectedExitCode,
			StderrContains: "TargetError",
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", reduced)
}
```

Exported API:

```go
func LoadJSON(path string) (any, error)

type FailureSpec struct {
	ExitCode       *int
	StdoutContains string
	StderrContains string
	ConfirmRuns    int
	Timeout        time.Duration
}

func VerifyOriginal(
	ctx context.Context,
	command string,
	candidate []byte,
	spec FailureSpec,
) error

func Reduce(
	ctx context.Context,
	original any,
	command string,
	spec FailureSpec,
) (any, error)
```

## Development checks

```sh
gofmt -w .
git diff --check
go vet ./...
go test -race ./...
```
