package reproslice

import "testing"

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
