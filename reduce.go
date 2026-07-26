package reproslice

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

func reduceObject(
	original map[string]any,
	reproduces func(map[string]any) (bool, error),
) (map[string]any, error) {
	current := cloneObject(original)

	for {
		before := current

		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			candidate := cloneObject(current)
			delete(candidate, key)

			ok, err := reproduces(candidate)
			if err != nil {
				return nil, err
			}
			if ok {
				current = candidate
			}
		}

		for _, key := range keys {
			child, exists := current[key]
			if !exists {
				continue
			}

			switch child := child.(type) {
			case map[string]any:
				reducedChild, err := reduceObject(
					child,
					func(candidate map[string]any) (bool, error) {
						candidateRoot := cloneObject(current)
						candidateRoot[key] = candidate
						return reproduces(candidateRoot)
					},
				)
				if err != nil {
					return nil, err
				}

				current = cloneObject(current)
				current[key] = reducedChild

			case []any:
				reducedChild, err := reduceArray(
					child,
					func(candidate []any) (bool, error) {
						candidateRoot := cloneObject(current)
						candidateRoot[key] = candidate
						return reproduces(candidateRoot)
					},
				)
				if err != nil {
					return nil, err
				}

				current = cloneObject(current)
				current[key] = reducedChild

			default:
				reducedChild, err := reduceScalar(
					child,
					func(candidate any) (bool, error) {
						candidateRoot := cloneObject(current)
						candidateRoot[key] = candidate
						return reproduces(candidateRoot)
					},
				)
				if err != nil {
					return nil, err
				}

				current = cloneObject(current)
				current[key] = reducedChild
			}
		}

		if reflect.DeepEqual(current, before) {
			return current, nil
		}
	}
}

func cloneObject(value map[string]any) map[string]any {
	clone := make(map[string]any, len(value))
	for key, item := range value {
		clone[key] = item
	}
	return clone
}

func reduceArray(
	original []any,
	reproduces func([]any) (bool, error),
) ([]any, error) {
	current := cloneArray(original)

	for {
		before := current

		for {
			removedChunk := false

			for chunkSize := len(current); chunkSize >= 2; chunkSize-- {
				for start := 0; start+chunkSize <= len(current); start++ {
					end := start + chunkSize

					candidate := make([]any, 0, len(current)-chunkSize)
					candidate = append(candidate, current[:start]...)
					candidate = append(candidate, current[end:]...)

					ok, err := reproduces(candidate)
					if err != nil {
						return nil, err
					}
					if ok {
						current = candidate
						removedChunk = true
						break
					}
				}

				if removedChunk {
					break
				}
			}

			if !removedChunk {
				break
			}
		}

		for index := 0; index < len(current); {
			candidate := make([]any, 0, len(current)-1)
			candidate = append(candidate, current[:index]...)
			candidate = append(candidate, current[index+1:]...)

			ok, err := reproduces(candidate)
			if err != nil {
				return nil, err
			}
			if ok {
				current = candidate
				continue
			}

			index++
		}

		for index, item := range current {
			switch child := item.(type) {
			case map[string]any:
				reducedChild, err := reduceObject(
					child,
					func(candidate map[string]any) (bool, error) {
						candidateRoot := cloneArray(current)
						candidateRoot[index] = candidate
						return reproduces(candidateRoot)
					},
				)
				if err != nil {
					return nil, err
				}

				current = cloneArray(current)
				current[index] = reducedChild

			case []any:
				reducedChild, err := reduceArray(
					child,
					func(candidate []any) (bool, error) {
						candidateRoot := cloneArray(current)
						candidateRoot[index] = candidate
						return reproduces(candidateRoot)
					},
				)
				if err != nil {
					return nil, err
				}

				current = cloneArray(current)
				current[index] = reducedChild

			default:
				reducedChild, err := reduceScalar(
					child,
					func(candidate any) (bool, error) {
						candidateRoot := cloneArray(current)
						candidateRoot[index] = candidate
						return reproduces(candidateRoot)
					},
				)
				if err != nil {
					return nil, err
				}

				current = cloneArray(current)
				current[index] = reducedChild
			}
		}

		if reflect.DeepEqual(current, before) {
			return current, nil
		}
	}
}

func cloneArray(value []any) []any {
	return append([]any(nil), value...)
}

func reduceScalar(
	original any,
	reproduces func(any) (bool, error),
) (any, error) {
	var candidates []any

	switch value := original.(type) {
	case string:
		if value != "" {
			candidates = append(candidates, "")
		}

		runes := []rune(value)
		if len(runes) > 1 {
			candidates = append(candidates, string(runes[:1]))
		}

	case json.Number:
		if value.String() != "0" {
			candidates = append(candidates, json.Number("0"))
		}
		if value.String() != "1" {
			candidates = append(candidates, json.Number("1"))
		}
		if value.String() != "-1" {
			candidates = append(candidates, json.Number("-1"))
		}

	case bool:
		if value {
			candidates = append(candidates, false)
		}

	case nil:
		return original, nil

	default:
		return original, nil
	}

	candidates = append(candidates, nil)

	for _, candidate := range candidates {
		reproducesFailure, err := reproduces(candidate)
		if err != nil {
			return nil, err
		}
		if reproducesFailure {
			return candidate, nil
		}
	}

	return original, nil
}

func reduceValue(
	original any,
	reproduces func(any) (bool, error),
) (any, error) {
	switch value := original.(type) {
	case map[string]any:
		return reduceObject(
			value,
			func(candidate map[string]any) (bool, error) {
				return reproduces(candidate)
			},
		)

	case []any:
		return reduceArray(
			value,
			func(candidate []any) (bool, error) {
				return reproduces(candidate)
			},
		)

	default:
		return reduceScalar(original, reproduces)
	}
}

func Reduce(
	ctx context.Context,
	original any,
	command string,
	spec FailureSpec,
) (any, error) {
	encodedOriginal, err := json.Marshal(original)
	if err != nil {
		return nil, fmt.Errorf("encode original payload: %w", err)
	}

	if err := VerifyOriginal(ctx, command, encodedOriginal, spec); err != nil {
		return nil, fmt.Errorf("verify original payload: %w", err)
	}

	return reduceValue(
		original,
		func(candidate any) (bool, error) {
			encodedCandidate, err := json.Marshal(candidate)
			if err != nil {
				return false, fmt.Errorf("encode candidate payload: %w", err)
			}

			return reproducesCommandFailure(
				ctx,
				command,
				encodedCandidate,
				spec,
			)
		},
	)
}

func reproducesCommandFailure(
	ctx context.Context,
	command string,
	candidate []byte,
	spec FailureSpec,
) (bool, error) {
	internalSpec, confirmRuns, err := normalizeFailureSpec(spec)
	if err != nil {
		return false, err
	}

	for run := 1; run <= confirmRuns; run++ {
		runCtx := ctx
		cancel := func() {}
		if spec.Timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, spec.Timeout)
		}

		result, err := runCommand(runCtx, command, candidate)
		cancel()
		if err != nil {
			return false, fmt.Errorf(
				"confirmation run %d of %d: run candidate payload: %w",
				run,
				confirmRuns,
				err,
			)
		}
		if !matchesFailure(result, internalSpec) {
			return false, nil
		}
	}

	return true, nil
}
