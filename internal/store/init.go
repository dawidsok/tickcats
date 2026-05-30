package store

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const gitignoreEntry = RootDir + "/"

func Init(root string) error {
	for _, state := range ValidStates {
		path := filepath.Join(root, StateDir(state))
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create state directory %q: %w", path, err)
		}
	}

	if err := ensureGitignoreEntry(filepath.Join(root, ".gitignore"), gitignoreEntry); err != nil {
		return fmt.Errorf("ensure .gitignore entry: %w", err)
	}

	return nil
}

func ensureGitignoreEntry(path string, entry string) error {
	lines, err := readLinesIfExists(path)
	if err != nil {
		return err
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	if len(lines) > 0 && lines[len(lines)-1] != "" {
		if _, err := file.WriteString("\n"); err != nil {
			return err
		}
	}

	_, err = file.WriteString(entry + "\n")
	return err
}

func readLinesIfExists(path string) ([]string, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
