package reproslice

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func LoadJSON(path string) (any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("input contains multiple JSON values")
		}
		return nil, err
	}

	return value, nil
}
