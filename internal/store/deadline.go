package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func SetDeadline(boardRoot string, name string, state State, deadline *time.Time) error {
	return setDeadlineAt(boardRoot, name, state, deadline, time.Now().UTC())
}

func setDeadlineAt(boardRoot string, name string, state State, deadline *time.Time, now time.Time) error {
	cleanName, err := validateTicketFilename(name)
	if err != nil {
		return err
	}
	path := filepath.Join(boardRoot, string(state), cleanName)
	data, _, err := readAndParseTicket(path)
	if err != nil {
		return err
	}
	rewritten, err := rewriteDeadlineFrontmatter(data, deadline, now.UTC())
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, rewritten, 0o644); err != nil {
		return fmt.Errorf("write ticket %q: %w", path, err)
	}
	return nil
}

func rewriteDeadlineFrontmatter(data []byte, deadline *time.Time, now time.Time) ([]byte, error) {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, fmt.Errorf("missing frontmatter opening fence")
	}
	rest := data[len("---\n"):]
	end := bytes.Index(rest, []byte("\n---\n"))
	if end < 0 {
		return nil, fmt.Errorf("missing frontmatter closing fence")
	}

	lines := strings.Split(string(rest[:end]), "\n")
	out := make([]string, 0, len(lines)+1)
	wroteDeadline := false
	for _, line := range lines {
		key, _, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && strings.TrimSpace(key) == "deadline" {
			continue
		}
		if ok && strings.TrimSpace(key) == "updated" {
			out = append(out, "updated: "+now.Format(time.RFC3339))
			if deadline != nil {
				out = append(out, "deadline: "+deadline.UTC().Format(time.DateOnly))
				wroteDeadline = true
			}
			continue
		}
		out = append(out, line)
	}
	if deadline != nil && !wroteDeadline {
		out = append(out, "deadline: "+deadline.UTC().Format(time.DateOnly))
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.WriteString(strings.Join(out, "\n"))
	buf.WriteString("\n---\n")
	buf.Write(rest[end+len("\n---\n"):])
	return buf.Bytes(), nil
}
