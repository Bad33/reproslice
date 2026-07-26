package reproslice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSONMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	_, err := LoadJSON(path)
	if err == nil {
		t.Fatal("LoadJSON() error = nil, want missing-file error")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("LoadJSON() error = %v, want error matching os.ErrNotExist", err)
	}
}

func TestLoadJSONValidObject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(`{"name":"reproslice"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadJSON(path)
	if err != nil {
		t.Fatalf("LoadJSON() error = %v", err)
	}

	object, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("LoadJSON() type = %T, want map[string]any", got)
	}
	if object["name"] != "reproslice" {
		t.Fatalf(`LoadJSON()["name"] = %v, want "reproslice"`, object["name"])
	}
}

func TestLoadJSONPreservesNumber(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(`9007199254740993`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadJSON(path)
	if err != nil {
		t.Fatalf("LoadJSON() error = %v", err)
	}

	number, ok := got.(json.Number)
	if !ok {
		t.Fatalf("LoadJSON() type = %T, want json.Number", got)
	}
	if number.String() != "9007199254740993" {
		t.Fatalf("LoadJSON() number = %q, want %q", number, "9007199254740993")
	}
}

func TestLoadJSONRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "malformed", input: `{"name":`},
		{name: "trailing content", input: `{"name":"reproslice"} trailing`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.json")
			if err := os.WriteFile(path, []byte(test.input), 0o600); err != nil {
				t.Fatal(err)
			}

			if _, err := LoadJSON(path); err == nil {
				t.Fatal("LoadJSON() error = nil, want invalid-JSON error")
			}
		})
	}
}
