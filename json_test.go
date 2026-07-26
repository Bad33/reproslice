package reproslice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSONMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	_, err := loadJSON(path)
	if err == nil {
		t.Fatal("loadJSON() error = nil, want missing-file error")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("loadJSON() error = %v, want error matching os.ErrNotExist", err)
	}
}

func TestLoadJSONValidObject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(`{"name":"reproslice"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := loadJSON(path)
	if err != nil {
		t.Fatalf("loadJSON() error = %v", err)
	}

	object, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("loadJSON() type = %T, want map[string]any", got)
	}
	if object["name"] != "reproslice" {
		t.Fatalf(`loadJSON()["name"] = %v, want "reproslice"`, object["name"])
	}
}

func TestLoadJSONPreservesNumber(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(`9007199254740993`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := loadJSON(path)
	if err != nil {
		t.Fatalf("loadJSON() error = %v", err)
	}

	number, ok := got.(json.Number)
	if !ok {
		t.Fatalf("loadJSON() type = %T, want json.Number", got)
	}
	if number.String() != "9007199254740993" {
		t.Fatalf("loadJSON() number = %q, want %q", number, "9007199254740993")
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

			if _, err := loadJSON(path); err == nil {
				t.Fatal("loadJSON() error = nil, want invalid-JSON error")
			}
		})
	}
}
