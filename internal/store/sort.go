// sort.go manages the active sort mode and manual ticket order for each column.
// SortConfig is persisted to "sort.json" so the user's sort preference and
// manual ordering survive TUI restarts.
// Manual sort stores an ordered list of ticket filenames per column; tickets
// not in the list (e.g. newly created) are appended at the end.
package store

type SortMode string

const (
	SortPriority SortMode = "priority"
	SortTitle    SortMode = "title"
	SortDate     SortMode = "date"
	SortManual   SortMode = "manual"
)

var SortModes = []SortMode{SortPriority, SortTitle, SortDate, SortManual}

type SortConfig struct {
	Mode        SortMode           `json:"mode"`
	ManualOrder map[State][]string `json:"manual_order,omitempty"`
}

func LoadSortConfig(boardRoot string) (SortConfig, error) {
	cfg, err := loadJSON(boardRoot, "sort.json", SortConfig{
		Mode:        SortPriority,
		ManualOrder: make(map[State][]string),
	})
	if err != nil {
		return cfg, err
	}
	if cfg.ManualOrder == nil {
		cfg.ManualOrder = make(map[State][]string)
	}
	return cfg, nil
}

func SaveSortConfig(boardRoot string, cfg SortConfig) error {
	return saveJSON(boardRoot, "sort.json", cfg)
}
