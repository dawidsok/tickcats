// json.go provides generic helpers for reading and writing JSON config files
// in the board root directory. loadJSON returns the caller-supplied default
// value when the file does not exist, allowing callers to define per-type
// defaults without duplicating the "file not found" handling.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func loadJSON[T any](boardRoot, filename string, defaultVal T) (T, error) {
	data, err := os.ReadFile(filepath.Join(boardRoot, filename))
	if os.IsNotExist(err) {
		return defaultVal, nil
	}
	if err != nil {
		return defaultVal, err
	}
	if err := json.Unmarshal(data, &defaultVal); err != nil {
		return defaultVal, err
	}
	return defaultVal, nil
}

func saveJSON[T any](boardRoot, filename string, val T) error {
	data, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(boardRoot, filename), data, 0o644)
}
