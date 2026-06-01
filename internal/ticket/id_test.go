package ticket

import "testing"

func TestValidID(t *testing.T) {
	tests := []struct {
		id string
		ok bool
	}{
		{id: "TC-A7K9Q2", ok: true},
		{id: "TC-AAAAAA", ok: true},
		{id: "tc-A7K9Q2", ok: false},
		{id: "TC-A7K9Q", ok: false},
		{id: "TC-A7K9Q20", ok: false},
		{id: "TC-A7K9O2", ok: false},
		{id: "XX-A7K9Q2", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := ValidID(tt.id); got != tt.ok {
				t.Fatalf("ValidID() = %v, want %v", got, tt.ok)
			}
		})
	}
}

func TestGenerateIDFormatAndAvoidsExisting(t *testing.T) {
	existing := map[string]bool{}
	id, err := GenerateID(existing)
	if err != nil {
		t.Fatalf("GenerateID() error = %v", err)
	}
	if !ValidID(id) {
		t.Fatalf("GenerateID() = %q, want valid ID", id)
	}
	existing[id] = true
	other, err := GenerateID(existing)
	if err != nil {
		t.Fatalf("GenerateID() second error = %v", err)
	}
	if other == id {
		t.Fatalf("GenerateID() returned existing ID %q", id)
	}
	if !ValidID(other) {
		t.Fatalf("GenerateID() second = %q, want valid ID", other)
	}
}
