// title.go implements structured title parsing.
// A ticket title has three optional parts in order: a label group "[blocked, to refine]",
// a kind prefix "Feat:" / "Bug:" / "Task:", and the plain text.
// ParseTitle decomposes these parts so callers can inspect kind and labels
// without re-parsing the raw string each time.
package ticket

import (
	"fmt"
	"strings"
)

// Kind is the ticket category: Feature, Task, or Bug.
// It controls the prefix shown in the normalized title (Feat:, Task:, Bug:).
type Kind string

const (
	KindFeature Kind = "feature"
	KindTask    Kind = "task"
	KindBug     Kind = "bug"
)

const (
	LabelBlocked  = "blocked"
	LabelToRefine = "to refine"
)

// ParsedTitle is the result of decomposing a raw ticket title string.
// HadPrefix indicates whether the original string included an explicit kind
// prefix (e.g. "Feat:") so callers can distinguish parsed vs. defaulted kind.
type ParsedTitle struct {
	Raw       string
	Labels    []string
	Kind      Kind
	Text      string
	HadPrefix bool
}

// ParseTitle decomposes a raw title string into its labels, kind, and text parts.
func ParseTitle(raw string) ParsedTitle {
	rest := strings.TrimSpace(raw)
	labels, rest := splitLabels(rest)

	kind, text, hadPrefix := splitKind(rest)
	return ParsedTitle{
		Raw:       raw,
		Labels:    labels,
		Kind:      kind,
		Text:      text,
		HadPrefix: hadPrefix,
	}
}

func (t ParsedTitle) HasLabel(label string) bool {
	needle := normalizeLabel(label)
	for _, got := range t.Labels {
		if got == needle {
			return true
		}
	}
	return false
}

func (t ParsedTitle) Blocked() bool {
	return t.HasLabel(LabelBlocked)
}

func (t ParsedTitle) ToRefine() bool {
	return t.HasLabel(LabelToRefine)
}

// NormalizedTitle rebuilds the canonical title string from the parsed parts,
// always including the kind prefix and any labels.
func (t ParsedTitle) NormalizedTitle() string {
	parts := make([]string, 0, 2)
	if len(t.Labels) > 0 {
		parts = append(parts, "["+strings.Join(t.Labels, ", ")+"]")
	}
	parts = append(parts, string(t.Kind.Prefix())+":")

	prefix := strings.Join(parts, " ")
	if t.Text == "" {
		return prefix
	}
	return prefix + " " + t.Text
}

func (k Kind) Prefix() string {
	switch k {
	case KindFeature:
		return "Feat"
	case KindBug:
		return "Bug"
	case KindTask:
		return "Task"
	default:
		return "Task"
	}
}

func splitLabels(raw string) ([]string, string) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "[") {
		return nil, trimmed
	}

	end := strings.Index(trimmed, "]")
	if end == -1 {
		return nil, trimmed
	}

	labelText := trimmed[1:end]
	parts := strings.Split(labelText, ",")
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		label := normalizeLabel(part)
		if label != "" {
			labels = append(labels, label)
		}
	}

	return labels, strings.TrimSpace(trimmed[end+1:])
}

func normalizeLabel(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// ParseKind parses a kind string from user input. Accepts the aliases "feat",
// "feature", "task", "bug", "fix". Returns an error for unrecognized values.
func ParseKind(raw string) (Kind, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "feat", "feature":
		return KindFeature, nil
	case "task":
		return KindTask, nil
	case "bug", "fix":
		return KindBug, nil
	default:
		return "", fmt.Errorf("unknown ticket kind %q", raw)
	}
}

func splitKind(raw string) (Kind, string, bool) {
	trimmed := strings.TrimSpace(raw)
	colon := strings.Index(trimmed, ":")
	if colon == -1 {
		return KindTask, trimmed, false
	}

	prefix := strings.ToLower(strings.TrimSpace(trimmed[:colon]))
	text := strings.TrimSpace(trimmed[colon+1:])

	switch prefix {
	case "feat", "feature":
		return KindFeature, text, true
	case "bug", "fix":
		return KindBug, text, true
	case "task":
		return KindTask, text, true
	default:
		return KindTask, trimmed, false
	}
}
