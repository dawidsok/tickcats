// config.go persists user-level board preferences (editor command, colour
// theme, skip-editor-prompt flag) to "config.json" in the board root.
// Missing file is treated as an empty config (all zero values / defaults).
package store

import (
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
