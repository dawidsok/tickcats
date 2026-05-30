package ticket

import "testing"

func TestParseTitle(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		wantKind      Kind
		wantText      string
		wantLabels    []string
		wantHadPrefix bool
	}{
		{
			name:          "feature prefix",
			raw:           "Feat: add import validation",
			wantKind:      KindFeature,
			wantText:      "add import validation",
			wantHadPrefix: true,
		},
		{
			name:          "bug prefix",
			raw:           "Bug: crash on empty backlog",
			wantKind:      KindBug,
			wantText:      "crash on empty backlog",
			wantHadPrefix: true,
		},
		{
			name:          "task prefix",
			raw:           "Task: clean up parser errors",
			wantKind:      KindTask,
			wantText:      "clean up parser errors",
			wantHadPrefix: true,
		},
		{
			name:     "missing prefix defaults to task",
			raw:      "write README",
			wantKind: KindTask,
			wantText: "write README",
		},
		{
			name:          "array labels before prefix",
			raw:           "[blocked, to refine] Feat: feature description",
			wantKind:      KindFeature,
			wantText:      "feature description",
			wantLabels:    []string{"blocked", "to refine"},
			wantHadPrefix: true,
		},
		{
			name:          "free-form idea label",
			raw:           "[idea, to refine] Feat: feature description",
			wantKind:      KindFeature,
			wantText:      "feature description",
			wantLabels:    []string{"idea", "to refine"},
			wantHadPrefix: true,
		},
		{
			name:          "label whitespace normalizes",
			raw:           "[ Idea , To Refine ] Feat: feature description",
			wantKind:      KindFeature,
			wantText:      "feature description",
			wantLabels:    []string{"idea", "to refine"},
			wantHadPrefix: true,
		},
		{
			name:          "empty label list ignored",
			raw:           "[] Feat: feature description",
			wantKind:      KindFeature,
			wantText:      "feature description",
			wantLabels:    nil,
			wantHadPrefix: true,
		},
		{
			name:     "unknown prefix stays task with full text",
			raw:      "Chore: rename files",
			wantKind: KindTask,
			wantText: "Chore: rename files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTitle(tt.raw)
			if got.Kind != tt.wantKind {
				t.Fatalf("Kind = %q, want %q", got.Kind, tt.wantKind)
			}
			if got.Text != tt.wantText {
				t.Fatalf("Text = %q, want %q", got.Text, tt.wantText)
			}
			if got.HadPrefix != tt.wantHadPrefix {
				t.Fatalf("HadPrefix = %v, want %v", got.HadPrefix, tt.wantHadPrefix)
			}
			if len(got.Labels) != len(tt.wantLabels) {
				t.Fatalf("Labels = %#v, want %#v", got.Labels, tt.wantLabels)
			}
			for i := range tt.wantLabels {
				if got.Labels[i] != tt.wantLabels[i] {
					t.Fatalf("Labels = %#v, want %#v", got.Labels, tt.wantLabels)
				}
			}
		})
	}
}

func TestParsedTitleSpecialLabels(t *testing.T) {
	title := ParseTitle("[blocked, to refine] Feat: feature description")
	if !title.Blocked() {
		t.Fatalf("Blocked() = false, want true")
	}
	if !title.ToRefine() {
		t.Fatalf("ToRefine() = false, want true")
	}
}

func TestParsedTitleNormalizedTitle(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "missing prefix adds task",
			raw:  "write README",
			want: "Task: write README",
		},
		{
			name: "missing prefix with labels adds task after labels",
			raw:  "[idea, to refine] write README",
			want: "[idea, to refine] Task: write README",
		},
		{
			name: "feature normalizes alias",
			raw:  "Feature: add import validation",
			want: "Feat: add import validation",
		},
		{
			name: "fix normalizes to bug",
			raw:  "[blocked] Fix: crash on empty backlog",
			want: "[blocked] Bug: crash on empty backlog",
		},
		{
			name: "multiple labels normalize to one bracket",
			raw:  "[blocked, to refine] Fix: crash on empty backlog",
			want: "[blocked, to refine] Bug: crash on empty backlog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTitle(tt.raw).NormalizedTitle()
			if got != tt.want {
				t.Fatalf("NormalizedTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}
