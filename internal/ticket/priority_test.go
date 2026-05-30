package ticket

import "testing"

func TestParsePriority(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want Priority
		ok   bool
	}{
		{name: "p0", raw: "P0", want: PriorityP0, ok: true},
		{name: "lowercase", raw: "p1", want: PriorityP1, ok: true},
		{name: "trim", raw: " P2 ", want: PriorityP2, ok: true},
		{name: "p3", raw: "P3", want: PriorityP3, ok: true},
		{name: "invalid", raw: "high", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePriority(tt.raw)
			if tt.ok && err != nil {
				t.Fatalf("ParsePriority() error = %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("ParsePriority() expected error")
			}
			if got != tt.want {
				t.Fatalf("ParsePriority() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPriorityHigherThan(t *testing.T) {
	if !PriorityP0.HigherThan(PriorityP1) {
		t.Fatalf("P0 should be higher than P1")
	}
	if PriorityP3.HigherThan(PriorityP2) {
		t.Fatalf("P3 should not be higher than P2")
	}
}
