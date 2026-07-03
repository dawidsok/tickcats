package store

import "testing"

func TestMatrixPrioritisationEnabledDefaultsOn(t *testing.T) {
	if !((Config{}).MatrixPrioritisationEnabled()) {
		t.Fatal("empty config should enable matrix prioritisation")
	}
}

func TestMatrixPrioritisationEnabledCanBeDisabled(t *testing.T) {
	cfg := Config{DisableMatrixPrioritisation: true}
	if cfg.MatrixPrioritisationEnabled() {
		t.Fatal("disabled config should disable matrix prioritisation")
	}
}
