// config.go persists user-level board preferences (editor command, colour
// theme, skip-editor-prompt flag) to "config.json" in the board root.
// Missing file is treated as an empty config (all zero values / defaults).
package store

// Config holds user preferences that are saved between TUI sessions.
// SkipEditorPrompt suppresses the "open in editor?" dialog after ticket creation.
type Config struct {
	Editor           string `json:"editor,omitempty"`
	Theme            int    `json:"theme,omitempty"`
	SkipEditorPrompt bool   `json:"skip_editor_prompt,omitempty"`
}

func LoadConfig(boardRoot string) (Config, error) {
	return loadJSON(boardRoot, "config.json", Config{})
}

func SaveConfig(boardRoot string, cfg Config) error {
	return saveJSON(boardRoot, "config.json", cfg)
}
