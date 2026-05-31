package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type SortMode string

const (
	SortPriority SortMode = "priority"
	SortTitle    SortMode = "title"
	SortDate     SortMode = "date"
	SortManual   SortMode = "manual"
)

var SortModes = []SortMode{SortPriority, SortTitle, SortDate, SortManual}

type SortConfig struct {
	Mode        SortMode            `json:"mode"`
	ManualOrder map[State][]string  `json:"manual_order,omitempty"`
}

func LoadSortConfig(boardRoot string) (SortConfig, error) {
	cfg := SortConfig{
		Mode:        SortPriority,
		ManualOrder: make(map[State][]string),
	}
	data, err := os.ReadFile(filepath.Join(boardRoot, "sort.json"))
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.ManualOrder == nil {
		cfg.ManualOrder = make(map[State][]string)
	}
	return cfg, nil
}

func SaveSortConfig(boardRoot string, cfg SortConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(boardRoot, "sort.json"), data, 0o644)
}
