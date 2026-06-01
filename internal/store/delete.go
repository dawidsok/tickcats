// delete.go implements soft-deletion by moving ticket files into a .trash
// subdirectory rather than permanently removing them. The source file is
// validated before the move to prevent trashing corrupted files.
package store

import (
	"fmt"
	"os"
	"path/filepath"
)

const TrashDir = ".trash"

// Trash moves a ticket file from its state directory into the .trash
// subdirectory. Returns the destination path on success.
func Trash(root string, name string, from State) (string, error) {
	if _, err := ParseState(string(from)); err != nil {
		return "", err
	}

	cleanName, err := validateTicketFilename(name)
	if err != nil {
		return "", err
	}

	source := filepath.Join(root, string(from), cleanName)
	if _, _, err := readAndParseTicket(source); err != nil {
		return "", err
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
