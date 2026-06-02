// config.go persists user-level board preferences (editor command, colour
// theme, skip-editor-prompt flag) to "config.json" in the board root.
// Missing file is treated as an empty config (all zero values / defaults).
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Column struct {
	ID          string `json:"id"`
	DisplayName string `json:"name"`
}

// Config holds user preferences that are saved between TUI sessions.
// SkipEditorPrompt suppresses the "open in editor?" dialog after ticket creation.
type Config struct {
	Editor           string   `json:"editor,omitempty"`
	Theme            int      `json:"theme,omitempty"`
	SkipEditorPrompt bool     `json:"skip_editor_prompt,omitempty"`
	Columns          []Column `json:"columns,omitempty"`
}

func DefaultColumns() []Column {
	return []Column{
		{ID: string(StateBacklog), DisplayName: "Backlog"},
		{ID: string(StateReady), DisplayName: "Ready"},
		{ID: string(StateDoing), DisplayName: "Doing"},
		{ID: string(StateDone), DisplayName: "Done"},
		{ID: string(StateWontDo), DisplayName: "Won't Do"},
	}
}

// GetColumns returns configured columns, falling back to the v1 default board
// columns when config.json has no column list yet.
func (c Config) GetColumns() []Column {
	if len(c.Columns) == 0 {
		return DefaultColumns()
	}
	cols := make([]Column, len(c.Columns))
	copy(cols, c.Columns)
	return cols
}

func LoadConfig(boardRoot string) (Config, error) {
	cfg, err := loadJSON(boardRoot, "config.json", Config{})
	if err != nil {
		return Config{}, err
	}

	synced, updated, err := SyncConfigColumns(boardRoot, cfg)
	if err != nil {
		return Config{}, err
	}
	if updated {
		if err := SaveConfig(boardRoot, synced); err != nil {
			return Config{}, err
		}
	}
	return synced, nil
}

func SaveConfig(boardRoot string, cfg Config) error {
	return saveJSON(boardRoot, "config.json", cfg)
}

// AddColumn creates a new column folder and appends it to config. The folder ID
// is derived from displayName; custom folder slugs are intentionally not part of
// this API slice.
func AddColumn(boardRoot string, displayName string) error {
	displayName = strings.TrimSpace(displayName)
	id := slugify(displayName)
	if id == "" {
		return fmt.Errorf("invalid column name %q", displayName)
	}

	cfg, err := LoadConfig(boardRoot)
	if err != nil {
		return err
	}
	columns := cfg.GetColumns()
	if columnIndex(columns, id) >= 0 {
		return fmt.Errorf("column %q already exists", id)
	}

	if err := os.MkdirAll(filepath.Join(boardRoot, id), 0o755); err != nil {
		return fmt.Errorf("create column folder %q: %w", id, err)
	}
	cfg.Columns = append(columns, Column{ID: id, DisplayName: displayName})
	return SaveConfig(boardRoot, cfg)
}

// RenameColumn renames a column folder and updates its config entry. The new
// folder ID is derived from newDisplayName.
func RenameColumn(boardRoot string, oldID string, newDisplayName string) error {
	newDisplayName = strings.TrimSpace(newDisplayName)
	newID := slugify(newDisplayName)
	if newID == "" {
		return fmt.Errorf("invalid column name %q", newDisplayName)
	}

	cfg, err := LoadConfig(boardRoot)
	if err != nil {
		return err
	}
	columns := cfg.GetColumns()
	idx := columnIndex(columns, oldID)
	if idx < 0 {
		return fmt.Errorf("column %q not found", oldID)
	}
	if existing := columnIndex(columns, newID); existing >= 0 && existing != idx {
		return fmt.Errorf("column %q already exists", newID)
	}

	oldDir := filepath.Join(boardRoot, oldID)
	newDir := filepath.Join(boardRoot, newID)
	if oldID != newID {
		if _, err := os.Stat(newDir); err == nil {
			return fmt.Errorf("target column folder %q already exists", newID)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check target column folder %q: %w", newID, err)
		}
		if err := os.Rename(oldDir, newDir); err != nil {
			return fmt.Errorf("rename column folder %q to %q: %w", oldID, newID, err)
		}
	}

	columns[idx] = Column{ID: newID, DisplayName: newDisplayName}
	cfg.Columns = columns
	return SaveConfig(boardRoot, cfg)
}

// ReorderColumns persists exactly the column order supplied by newOrder. The
// input must contain each configured column ID once.
func ReorderColumns(boardRoot string, newOrder []string) error {
	cfg, err := LoadConfig(boardRoot)
	if err != nil {
		return err
	}
	columns := cfg.GetColumns()
	if len(newOrder) != len(columns) {
		return fmt.Errorf("new column order has %d columns, want %d", len(newOrder), len(columns))
	}

	byID := make(map[string]Column, len(columns))
	for _, col := range columns {
		byID[col.ID] = col
	}
	seen := make(map[string]bool, len(newOrder))
	reordered := make([]Column, 0, len(newOrder))
	for _, id := range newOrder {
		col, ok := byID[id]
		if !ok {
			return fmt.Errorf("unknown column %q", id)
		}
		if seen[id] {
			return fmt.Errorf("duplicate column %q", id)
		}
		seen[id] = true
		reordered = append(reordered, col)
	}

	cfg.Columns = reordered
	return SaveConfig(boardRoot, cfg)
}

// DeleteColumn removes a non-first column. Tickets in that column are moved to
// the first configured column before the folder and config entry are removed.
func DeleteColumn(boardRoot string, id string) error {
	cfg, err := LoadConfig(boardRoot)
	if err != nil {
		return err
	}
	columns := cfg.GetColumns()
	idx := columnIndex(columns, id)
	if idx < 0 {
		return fmt.Errorf("column %q not found", id)
	}
	if idx == 0 {
		return fmt.Errorf("the first column (%s) cannot be deleted", columns[0].DisplayName)
	}

	recipientID := columns[0].ID
	sourceDir := filepath.Join(boardRoot, id)
	recipientDir := filepath.Join(boardRoot, recipientID)
	entries, err := os.ReadDir(sourceDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read column folder %q: %w", id, err)
	}
	if err := os.MkdirAll(recipientDir, 0o755); err != nil {
		return fmt.Errorf("create recipient column folder %q: %w", recipientID, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		source := filepath.Join(sourceDir, entry.Name())
		target := filepath.Join(recipientDir, entry.Name())
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("target ticket already exists %q", target)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check target ticket %q: %w", target, err)
		}
		if err := os.Rename(source, target); err != nil {
			return fmt.Errorf("move ticket %q to %q: %w", source, target, err)
		}
	}

	if err := os.RemoveAll(sourceDir); err != nil {
		return fmt.Errorf("remove column folder %q: %w", id, err)
	}
	cfg.Columns = append(columns[:idx], columns[idx+1:]...)
	return SaveConfig(boardRoot, cfg)
}

func columnIndex(columns []Column, id string) int {
	for i, col := range columns {
		if col.ID == id {
			return i
		}
	}
	return -1
}

// SyncConfigColumns reconciles config columns with folders on disk.
//
//   - folders on disk but not in config are appended to config,
//   - columns in config whose folders are missing are removed,
//   - hidden/system folders are ignored,
//   - missing folders are not recreated just because config mentions them.
func SyncConfigColumns(boardRoot string, cfg Config) (Config, bool, error) {
	entries, err := os.ReadDir(boardRoot)
	if os.IsNotExist(err) {
		return cfg, false, nil
	}
	if err != nil {
		return cfg, false, err
	}

	diskFolders := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() || ignoredColumnFolder(entry.Name()) {
			continue
		}
		diskFolders[entry.Name()] = true
	}

	// If the board root exists but has no column folders yet, leave config alone.
	// This avoids converting an uninitialised directory into an empty board and
	// still honours the init flow, which creates the default folders explicitly.
	if len(diskFolders) == 0 {
		return cfg, false, nil
	}

	configured := cfg.GetColumns()
	synced := make([]Column, 0, len(configured)+len(diskFolders))
	included := make(map[string]bool)
	changed := len(cfg.Columns) == 0

	for _, col := range configured {
		if diskFolders[col.ID] {
			synced = append(synced, col)
			included[col.ID] = true
			continue
		}
		changed = true
	}

	for _, entry := range entries {
		if !entry.IsDir() || ignoredColumnFolder(entry.Name()) {
			continue
		}
		id := entry.Name()
		if included[id] {
			continue
		}
		synced = append(synced, Column{ID: id, DisplayName: formatDisplayName(id)})
		included[id] = true
		changed = true
	}

	if !changed {
		return cfg, false, nil
	}
	cfg.Columns = synced
	return cfg, true, nil
}

func ignoredColumnFolder(name string) bool {
	return name == ".trash" || name == ".git" || strings.HasPrefix(name, ".")
}

func formatDisplayName(id string) string {
	switch id {
	case string(StateBacklog):
		return "Backlog"
	case string(StateReady):
		return "Ready"
	case string(StateDoing):
		return "Doing"
	case string(StateDone):
		return "Done"
	case string(StateWontDo):
		return "Won't Do"
	}

	parts := strings.FieldsFunc(filepath.ToSlash(id), func(r rune) bool {
		return r == '-' || r == '_' || r == '/' || r == ' '
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}
