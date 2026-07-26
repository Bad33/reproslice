package reproslice

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestReduceExecutesEachSerializedCandidateOnce(t *testing.T) {
	expectedExitCode := 7
	runsPath := filepath.Join(t.TempDir(), "runs.log")
	quotedRunsPath := strconv.Quote(runsPath)

	command := `cat {input} >> ` + quotedRunsPath +
		`; printf '\n' >> ` + quotedRunsPath +
		`; if grep -q '"required":"keep"' {input}; then exit 7; fi`

	_, err := Reduce(
		t.Context(),
		map[string]any{
			"required": "keep",
			"noise":    "drop",
		},
		command,
		FailureSpec{ExitCode: &expectedExitCode},
	)
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	data, err := os.ReadFile(runsPath)
	if err != nil {
		t.Fatalf("read command execution log: %v", err)
	}

	counts := make(map[string]int)
	for _, candidate := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		counts[candidate]++
	}

	var duplicates []string
	for candidate, count := range counts {
		if count > 1 {
			duplicates = append(
				duplicates,
				fmt.Sprintf("%s executed %d times", candidate, count),
			)
		}
	}
	sort.Strings(duplicates)

	if len(duplicates) > 0 {
		t.Fatalf(
			"identical serialized candidates were executed repeatedly: %s",
			strings.Join(duplicates, "; "),
		)
	}
}
