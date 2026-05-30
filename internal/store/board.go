package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dawidsok/tickcats/internal/ticket"
)

type Board struct {
	Columns  map[State][]StoredTicket
	Warnings []Warning
}

type StoredTicket struct {
	Path   string
	Name   string
	State  State
	Ticket ticket.Ticket
}

type Warning struct {
	Path string
	Err  error
}

func LoadBoard(root string) (Board, error) {
	board := Board{
		Columns:  make(map[State][]StoredTicket, len(ValidStates)),
		Warnings: make([]Warning, 0),
	}
	for _, state := range ValidStates {
		board.Columns[state] = []StoredTicket{}
	}

	for _, state := range ValidStates {
		dir := filepath.Join(root, StateDir(state))
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return Board{}, fmt.Errorf("read state directory %q: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				board.Warnings = append(board.Warnings, Warning{Path: path, Err: err})
				continue
			}

			parsed, err := ticket.ParseMarkdown(data)
			if err != nil {
				board.Warnings = append(board.Warnings, Warning{Path: path, Err: err})
				continue
			}

			board.Columns[state] = append(board.Columns[state], StoredTicket{
				Path:   path,
				Name:   entry.Name(),
				State:  state,
				Ticket: parsed,
			})
		}

		sort.Slice(board.Columns[state], func(i, j int) bool {
			return board.Columns[state][i].Name < board.Columns[state][j].Name
		})
	}

	return board, nil
}

func Move(root string, name string, from State, to State) (string, error) {
	if _, err := ParseState(string(from)); err != nil {
		return "", err
	}
	if _, err := ParseState(string(to)); err != nil {
		return "", err
	}

	cleanName := filepath.Base(name)
	if cleanName != name {
		return "", fmt.Errorf("ticket name must be a file name, got %q", name)
	}
	if !strings.HasSuffix(cleanName, ".md") {
		return "", fmt.Errorf("ticket name must end with .md, got %q", name)
	}

	source := filepath.Join(root, StateDir(from), cleanName)
	targetDir := filepath.Join(root, StateDir(to))
	target := filepath.Join(targetDir, cleanName)

	data, err := os.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("read source ticket %q: %w", source, err)
	}
	if _, err := ticket.ParseMarkdown(data); err != nil {
		return "", fmt.Errorf("parse source ticket %q: %w", source, err)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("create target directory %q: %w", targetDir, err)
	}
	if _, err := os.Stat(target); err == nil {
		return "", fmt.Errorf("target ticket already exists %q", target)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("check target ticket %q: %w", target, err)
	}

	if err := os.Rename(source, target); err != nil {
		return "", fmt.Errorf("move ticket %q to %q: %w", source, target, err)
	}
	return target, nil
}
