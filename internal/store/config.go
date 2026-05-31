package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Editor           string `json:"editor,omitempty"`
	Theme            int    `json:"theme,omitempty"`
	SkipEditorPrompt bool   `json:"skip_editor_prompt,omitempty"`
}

func LoadConfig(boardRoot string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(filepath.Join(boardRoot, "config.json"))
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

func SaveConfig(boardRoot string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(boardRoot, "config.json"), data, 0o644)
}
