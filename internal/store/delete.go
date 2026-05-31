package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dawidsok/tickcats/internal/ticket"
)

const TrashDir = ".trash"

func Trash(root string, name string, from State) (string, error) {
	if _, err := ParseState(string(from)); err != nil {
		return "", err
	}

	cleanName := filepath.Base(name)
	if cleanName != name {
		return "", fmt.Errorf("ticket name must be a file name, got %q", name)
	}
	if !strings.HasSuffix(cleanName, ".md") {
		return "", fmt.Errorf("ticket name must end with .md, got %q", name)
	}

	source := filepath.Join(root, string(from), cleanName)
	data, err := os.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("read source ticket %q: %w", source, err)
	}
	if _, err := ticket.ParseMarkdown(data); err != nil {
		return "", fmt.Errorf("parse source ticket %q: %w", source, err)
	}

	trashDir := filepath.Join(root, TrashDir)
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		return "", fmt.Errorf("create trash directory %q: %w", trashDir, err)
	}

	target := filepath.Join(trashDir, cleanName)
	if _, err := os.Stat(target); err == nil {
		return "", fmt.Errorf("trash ticket already exists %q", target)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("check trash ticket %q: %w", target, err)
	}

	if err := os.Rename(source, target); err != nil {
		return "", fmt.Errorf("trash ticket %q to %q: %w", source, target, err)
	}
	return target, nil
}
