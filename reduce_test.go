package reproslice

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestReduceObjectRemovesUnneededField(t *testing.T) {
	original := map[string]any{
		"required": "keep",
		"noise":    "drop",
	}

	got, err := reduceObject(
		original,
		func(candidate map[string]any) (bool, error) {
			return candidate["required"] == "keep", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceObject() error = %v", err)
	}

	want := map[string]any{
		"required": "keep",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceObject() = %#v, want %#v", got, want)
	}

	wantOriginal := map[string]any{
		"required": "keep",
		"noise":    "drop",
	}
	if !reflect.DeepEqual(original, wantOriginal) {
		t.Fatalf("original = %#v, want unchanged %#v", original, wantOriginal)
	}
}

func TestReduceObjectReducesNestedObject(t *testing.T) {
	original := map[string]any{
		"payload": map[string]any{
			"required": "keep",
			"noise":    "drop",
		},
	}

	got, err := reduceObject(
		original,
		func(candidate map[string]any) (bool, error) {
			payload, ok := candidate["payload"].(map[string]any)
			if !ok {
				return false, nil
			}
			return payload["required"] == "keep", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceObject() error = %v", err)
	}

	want := map[string]any{
		"payload": map[string]any{
			"required": "keep",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceObject() = %#v, want %#v", got, want)
	}

	wantOriginal := map[string]any{
		"payload": map[string]any{
			"required": "keep",
			"noise":    "drop",
		},
	}
	if !reflect.DeepEqual(original, wantOriginal) {
		t.Fatalf("original = %#v, want unchanged %#v", original, wantOriginal)
	}
}

func TestReduceArrayRemovesUnneededElements(t *testing.T) {
	original := []any{
		"noise-before",
		"required",
		"noise-after",
	}

	got, err := reduceArray(
		original,
		func(candidate []any) (bool, error) {
			for _, item := range candidate {
				if item == "required" {
					return true, nil
				}
			}
			return false, nil
		},
	)
	if err != nil {
		t.Fatalf("reduceArray() error = %v", err)
	}

	want := []any{"required"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceArray() = %#v, want %#v", got, want)
	}

	wantOriginal := []any{
		"noise-before",
		"required",
		"noise-after",
	}
	if !reflect.DeepEqual(original, wantOriginal) {
		t.Fatalf("original = %#v, want unchanged %#v", original, wantOriginal)
	}
}

func TestReduceArrayReducesNestedObject(t *testing.T) {
	original := []any{
		map[string]any{
			"required": "keep",
			"noise":    "drop",
		},
	}

	got, err := reduceArray(
		original,
		func(candidate []any) (bool, error) {
			if len(candidate) != 1 {
				return false, nil
			}

			object, ok := candidate[0].(map[string]any)
			if !ok {
				return false, nil
			}
			return object["required"] == "keep", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceArray() error = %v", err)
	}

	want := []any{
		map[string]any{
			"required": "keep",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceArray() = %#v, want %#v", got, want)
	}

	wantOriginal := []any{
		map[string]any{
			"required": "keep",
			"noise":    "drop",
		},
	}
	if !reflect.DeepEqual(original, wantOriginal) {
		t.Fatalf("original = %#v, want unchanged %#v", original, wantOriginal)
	}
}

func TestReduceArrayReducesNestedArray(t *testing.T) {
	original := []any{
		[]any{
			"noise",
			"required",
		},
	}

	got, err := reduceArray(
		original,
		func(candidate []any) (bool, error) {
			if len(candidate) != 1 {
				return false, nil
			}

			child, ok := candidate[0].([]any)
			if !ok {
				return false, nil
			}

			for _, item := range child {
				if item == "required" {
					return true, nil
				}
			}
			return false, nil
		},
	)
	if err != nil {
		t.Fatalf("reduceArray() error = %v", err)
	}

	want := []any{
		[]any{"required"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceArray() = %#v, want %#v", got, want)
	}

	wantOriginal := []any{
		[]any{
			"noise",
			"required",
		},
	}
	if !reflect.DeepEqual(original, wantOriginal) {
		t.Fatalf("original = %#v, want unchanged %#v", original, wantOriginal)
	}
}

func TestReduceObjectReducesNestedArray(t *testing.T) {
	original := map[string]any{
		"items": []any{
			"noise",
			"required",
		},
	}

	got, err := reduceObject(
		original,
		func(candidate map[string]any) (bool, error) {
			items, ok := candidate["items"].([]any)
			if !ok {
				return false, nil
			}

			for _, item := range items {
				if item == "required" {
					return true, nil
				}
			}
			return false, nil
		},
	)
	if err != nil {
		t.Fatalf("reduceObject() error = %v", err)
	}

	want := map[string]any{
		"items": []any{"required"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceObject() = %#v, want %#v", got, want)
	}

	wantOriginal := map[string]any{
		"items": []any{
			"noise",
			"required",
		},
	}
	if !reflect.DeepEqual(original, wantOriginal) {
		t.Fatalf("original = %#v, want unchanged %#v", original, wantOriginal)
	}
}

func TestReduceScalarSimplifiesString(t *testing.T) {
	got, err := reduceScalar(
		"verbose diagnostic text",
		func(candidate any) (bool, error) {
			return candidate == "", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceScalar() error = %v", err)
	}
	if got != "" {
		t.Fatalf("reduceScalar() = %#v, want empty string", got)
	}
}

func TestReduceScalarSimplifiesNumberToZero(t *testing.T) {
	got, err := reduceScalar(
		json.Number("9007199254740993"),
		func(candidate any) (bool, error) {
			number, ok := candidate.(json.Number)
			return ok && number.String() == "0", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceScalar() error = %v", err)
	}

	want := json.Number("0")
	if got != want {
		t.Fatalf("reduceScalar() = %#v, want %#v", got, want)
	}
}

func TestReduceScalarSimplifiesBooleanToFalse(t *testing.T) {
	got, err := reduceScalar(
		true,
		func(candidate any) (bool, error) {
			value, ok := candidate.(bool)
			return ok && !value, nil
		},
	)
	if err != nil {
		t.Fatalf("reduceScalar() error = %v", err)
	}
	if got != false {
		t.Fatalf("reduceScalar() = %#v, want false", got)
	}
}

func TestReduceScalarFallsBackToNull(t *testing.T) {
	got, err := reduceScalar(
		"verbose diagnostic text",
		func(candidate any) (bool, error) {
			return candidate == nil, nil
		},
	)
	if err != nil {
		t.Fatalf("reduceScalar() error = %v", err)
	}
	if got != nil {
		t.Fatalf("reduceScalar() = %#v, want nil", got)
	}
}

func TestReduceObjectSimplifiesScalarField(t *testing.T) {
	original := map[string]any{
		"message": "verbose diagnostic text",
	}

	got, err := reduceObject(
		original,
		func(candidate map[string]any) (bool, error) {
			value, exists := candidate["message"]
			if !exists {
				return false, nil
			}
			return value == "", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceObject() error = %v", err)
	}

	want := map[string]any{
		"message": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceObject() = %#v, want %#v", got, want)
	}

	wantOriginal := map[string]any{
		"message": "verbose diagnostic text",
	}
	if !reflect.DeepEqual(original, wantOriginal) {
		t.Fatalf("original = %#v, want unchanged %#v", original, wantOriginal)
	}
}

func TestReduceArraySimplifiesScalarElement(t *testing.T) {
	original := []any{
		"verbose diagnostic text",
	}

	got, err := reduceArray(
		original,
		func(candidate []any) (bool, error) {
			return len(candidate) == 1 && candidate[0] == "", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceArray() error = %v", err)
	}

	want := []any{""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceArray() = %#v, want %#v", got, want)
	}

	wantOriginal := []any{
		"verbose diagnostic text",
	}
	if !reflect.DeepEqual(original, wantOriginal) {
		t.Fatalf("original = %#v, want unchanged %#v", original, wantOriginal)
	}
}

func TestReduceValueDispatchesObject(t *testing.T) {
	original := any(map[string]any{
		"required": "keep",
		"noise":    "drop",
	})

	got, err := reduceValue(
		original,
		func(candidate any) (bool, error) {
			object, ok := candidate.(map[string]any)
			if !ok {
				return false, nil
			}
			return object["required"] == "keep", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceValue() error = %v", err)
	}

	want := any(map[string]any{
		"required": "keep",
	})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceValue() = %#v, want %#v", got, want)
	}
}

func TestReduceValueDispatchesArray(t *testing.T) {
	original := any([]any{
		"noise",
		"required",
	})

	got, err := reduceValue(
		original,
		func(candidate any) (bool, error) {
			array, ok := candidate.([]any)
			if !ok {
				return false, nil
			}

			for _, item := range array {
				if item == "required" {
					return true, nil
				}
			}
			return false, nil
		},
	)
	if err != nil {
		t.Fatalf("reduceValue() error = %v", err)
	}

	want := any([]any{"required"})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceValue() = %#v, want %#v", got, want)
	}
}

func TestReduceValueDispatchesScalar(t *testing.T) {
	got, err := reduceValue(
		"verbose diagnostic text",
		func(candidate any) (bool, error) {
			return candidate == "", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceValue() error = %v", err)
	}
	if got != "" {
		t.Fatalf("reduceValue() = %#v, want empty string", got)
	}
}

func TestReduceMinimizesCommandFailure(t *testing.T) {
	expectedExitCode := 7
	original := map[string]any{
		"required": "keep",
		"noise":    "drop",
	}

	got, err := Reduce(
		t.Context(),
		original,
		`if grep -q '"required":"keep"' {input}; then printf "TargetError" >&2; exit 7; fi`,
		FailureSpec{
			ExitCode:       &expectedExitCode,
			StderrContains: "TargetError",
		},
	)
	if err != nil {
		t.Fatalf("Reduce() error = %v", err)
	}

	want := map[string]any{
		"required": "keep",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Reduce() = %#v, want %#v", got, want)
	}

	wantOriginal := map[string]any{
		"required": "keep",
		"noise":    "drop",
	}
	if !reflect.DeepEqual(original, wantOriginal) {
		t.Fatalf("original = %#v, want unchanged %#v", original, wantOriginal)
	}
}

func TestReduceObjectRepeatsUntilNoMoreChanges(t *testing.T) {
	original := map[string]any{
		"guard":   true,
		"message": "verbose",
	}

	got, err := reduceObject(
		original,
		func(candidate map[string]any) (bool, error) {
			message, exists := candidate["message"]
			if !exists {
				return false, nil
			}
			if message == "" {
				return true, nil
			}

			guard, exists := candidate["guard"]
			return exists && guard == true && message == "verbose", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceObject() error = %v", err)
	}

	want := map[string]any{
		"message": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceObject() = %#v, want %#v", got, want)
	}
}

func TestReduceArrayRepeatsUntilNoMoreChanges(t *testing.T) {
	original := []any{
		true,
		"verbose",
	}

	got, err := reduceArray(
		original,
		func(candidate []any) (bool, error) {
			if len(candidate) == 1 && candidate[0] == "" {
				return true, nil
			}
			if len(candidate) != 2 || candidate[0] != true {
				return false, nil
			}

			return candidate[1] == "verbose" || candidate[1] == "", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceArray() error = %v", err)
	}

	want := []any{""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceArray() = %#v, want %#v", got, want)
	}
}

func TestReduceArrayRemovesChunkWhenSingleDeletionFails(t *testing.T) {
	original := []any{
		"a",
		"b",
		"required",
	}

	got, err := reduceArray(
		original,
		func(candidate []any) (bool, error) {
			var hasA, hasB, hasRequired bool
			for _, item := range candidate {
				switch item {
				case "a":
					hasA = true
				case "b":
					hasB = true
				case "required":
					hasRequired = true
				}
			}

			return hasRequired && hasA == hasB, nil
		},
	)
	if err != nil {
		t.Fatalf("reduceArray() error = %v", err)
	}

	want := []any{"required"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reduceArray() = %#v, want %#v", got, want)
	}
}

func TestReduceScalarSimplifiesNumberToOne(t *testing.T) {
	original := json.Number("42")

	got, err := reduceScalar(
		original,
		func(candidate any) (bool, error) {
			number, ok := candidate.(json.Number)
			return ok && number.String() == "1", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceScalar() error = %v", err)
	}

	want := json.Number("1")
	if got != want {
		t.Fatalf("reduceScalar() = %#v, want %#v", got, want)
	}
}

func TestReduceScalarSimplifiesNumberToNegativeOne(t *testing.T) {
	original := json.Number("42")

	got, err := reduceScalar(
		original,
		func(candidate any) (bool, error) {
			number, ok := candidate.(json.Number)
			return ok && number.String() == "-1", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceScalar() error = %v", err)
	}

	want := json.Number("-1")
	if got != want {
		t.Fatalf("reduceScalar() = %#v, want %#v", got, want)
	}
}

func TestReduceScalarSimplifiesStringToFirstCharacter(t *testing.T) {
	original := "verbose"

	got, err := reduceScalar(
		original,
		func(candidate any) (bool, error) {
			value, ok := candidate.(string)
			return ok && value == "v", nil
		},
	)
	if err != nil {
		t.Fatalf("reduceScalar() error = %v", err)
	}

	if got != "v" {
		t.Fatalf("reduceScalar() = %#v, want %q", got, "v")
	}
}
