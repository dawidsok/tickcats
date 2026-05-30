package ticket

import "strings"

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

type ParsedTitle struct {
	Raw       string
	Labels    []string
	Kind      Kind
	Text      string
	HadPrefix bool
}

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
