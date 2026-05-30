package store

import "testing"

func TestParseState(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want State
		ok   bool
	}{
		{name: "backlog", raw: "backlog", want: StateBacklog, ok: true},
		{name: "ready", raw: "ready", want: StateReady, ok: true},
		{name: "doing", raw: "doing", want: StateDoing, ok: true},
		{name: "done", raw: "done", want: StateDone, ok: true},
		{name: "trim and lowercase", raw: " Ready ", want: StateReady, ok: true},
		{name: "invalid", raw: "blocked", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseState(tt.raw)
			if tt.ok && err != nil {
				t.Fatalf("ParseState() error = %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("ParseState() expected error")
			}
			if got != tt.want {
				t.Fatalf("ParseState() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStateDir(t *testing.T) {
	got := StateDir(StateReady)
	want := ".tickcats/ready"
	if got != want {
		t.Fatalf("StateDir() = %q, want %q", got, want)
	}
}
