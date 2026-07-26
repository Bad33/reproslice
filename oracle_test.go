package reproslice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMatchesFailureByExitCode(t *testing.T) {
	expected := 7

	tests := []struct {
		name   string
		result commandResult
		want   bool
	}{
		{
			name:   "matching exit code",
			result: commandResult{exitCode: 7},
			want:   true,
		},
		{
			name:   "different exit code",
			result: commandResult{exitCode: 8},
			want:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := failureSpec{exitCode: &expected}

			if got := matchesFailure(test.result, spec); got != test.want {
				t.Fatalf("matchesFailure() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestMatchesFailureByExitCodeAndStandardOutput(t *testing.T) {
	expectedExitCode := 7

	tests := []struct {
		name   string
		result commandResult
		want   bool
	}{
		{
			name: "all matchers pass",
			result: commandResult{
				exitCode: 7,
				stdout:   []byte("validation failed: bad discount"),
			},
			want: true,
		},
		{
			name: "stdout text absent",
			result: commandResult{
				exitCode: 7,
				stdout:   []byte("different failure"),
			},
			want: false,
		},
		{
			name: "exit code differs",
			result: commandResult{
				exitCode: 8,
				stdout:   []byte("validation failed: bad discount"),
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := failureSpec{
				exitCode:       &expectedExitCode,
				stdoutContains: "bad discount",
			}

			if got := matchesFailure(test.result, spec); got != test.want {
				t.Fatalf("matchesFailure() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestMatchesFailureByExitCodeAndOutputText(t *testing.T) {
	expectedExitCode := 7

	tests := []struct {
		name   string
		result commandResult
		want   bool
	}{
		{
			name: "all matchers pass",
			result: commandResult{
				exitCode: 7,
				stdout:   []byte("request rejected"),
				stderr:   []byte("DiscountValueError: invalid discount"),
			},
			want: true,
		},
		{
			name: "stderr text absent",
			result: commandResult{
				exitCode: 7,
				stdout:   []byte("request rejected"),
				stderr:   []byte("different failure"),
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := failureSpec{
				exitCode:       &expectedExitCode,
				stdoutContains: "request rejected",
				stderrContains: "DiscountValueError",
			}

			if got := matchesFailure(test.result, spec); got != test.want {
				t.Fatalf("matchesFailure() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestValidateFailureSpecRequiresMatcher(t *testing.T) {
	if err := validateFailureSpec(failureSpec{}); err == nil {
		t.Fatal("validateFailureSpec() error = nil, want missing-matcher error")
	}

	if err := validateFailureSpec(failureSpec{stderrContains: "DiscountValueError"}); err != nil {
		t.Fatalf("validateFailureSpec() error = %v, want nil", err)
	}
}

func TestVerifyOriginalRejectsNonMatchingFailure(t *testing.T) {
	expectedExitCode := 7

	err := verifyOriginal(
		t.Context(),
		`printf "DifferentError" >&2; test -f {input}; exit 7`,
		[]byte(`{}`),
		failureSpec{
			exitCode:       &expectedExitCode,
			stderrContains: "DiscountValueError",
		},
	)
	if err == nil {
		t.Fatal("verifyOriginal() error = nil, want non-reproducing-input error")
	}
}

func TestVerifyOriginalAcceptsMatchingFailure(t *testing.T) {
	expectedExitCode := 7

	err := verifyOriginal(
		t.Context(),
		`printf "DiscountValueError" >&2; test -f {input}; exit 7`,
		[]byte(`{"discount":-1}`),
		failureSpec{
			exitCode:       &expectedExitCode,
			stderrContains: "DiscountValueError",
		},
	)
	if err != nil {
		t.Fatalf("verifyOriginal() error = %v, want nil", err)
	}
}

func TestVerifyOriginalRunsRequiredConfirmations(t *testing.T) {
	counterPath := filepath.Join(t.TempDir(), "count")
	expectedExitCode := 7

	command := `count=$(cat ` + shellQuote(counterPath) + ` 2>/dev/null || echo 0); ` +
		`count=$((count + 1)); printf %s "$count" > ` + shellQuote(counterPath) + `; ` +
		`printf "DiscountValueError" >&2; test -f {input}; exit 7`

	err := VerifyOriginal(
		t.Context(),
		command,
		[]byte(`{}`),
		FailureSpec{
			ExitCode:       &expectedExitCode,
			StderrContains: "DiscountValueError",
			ConfirmRuns:    3,
		},
	)
	if err != nil {
		t.Fatalf("VerifyOriginal() error = %v, want nil", err)
	}

	got, err := os.ReadFile(counterPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "3" {
		t.Fatalf("command runs = %q, want %q", got, "3")
	}
}

func TestVerifyOriginalDefaultsToOneConfirmation(t *testing.T) {
	counterPath := filepath.Join(t.TempDir(), "count")
	expectedExitCode := 7

	command := `printf x >> ` + shellQuote(counterPath) + `; test -f {input}; exit 7`

	err := VerifyOriginal(
		t.Context(),
		command,
		[]byte(`{}`),
		FailureSpec{ExitCode: &expectedExitCode},
	)
	if err != nil {
		t.Fatalf("VerifyOriginal() error = %v, want nil", err)
	}

	got, err := os.ReadFile(counterPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "x" {
		t.Fatalf("command runs = %q, want one run", got)
	}
}

func TestVerifyOriginalRejectsInvalidExecutionSettings(t *testing.T) {
	expectedExitCode := 7

	tests := []struct {
		name    string
		spec    FailureSpec
		wantErr string
	}{
		{
			name: "negative confirmation runs",
			spec: FailureSpec{
				ExitCode:    &expectedExitCode,
				ConfirmRuns: -1,
			},
			wantErr: "confirmation runs",
		},
		{
			name: "negative timeout",
			spec: FailureSpec{
				ExitCode: &expectedExitCode,
				Timeout:  -time.Second,
			},
			wantErr: "timeout",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := VerifyOriginal(
				t.Context(),
				`test -f {input}; exit 7`,
				[]byte(`{}`),
				test.spec,
			)
			if err == nil {
				t.Fatal("VerifyOriginal() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf(
					"VerifyOriginal() error = %q, want message containing %q",
					err,
					test.wantErr,
				)
			}
		})
	}
}
