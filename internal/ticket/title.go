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
	labels := make([]string, 0)

	for {
		if !strings.HasPrefix(rest, "[") {
			break
		}
		end := strings.Index(rest, "]")
		if end == -1 {
			break
		}

		label := strings.ToLower(strings.TrimSpace(rest[1:end]))
		if label != "" {
			labels = append(labels, label)
		}
		rest = strings.TrimSpace(rest[end+1:])
	}

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
	needle := strings.ToLower(strings.TrimSpace(label))
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
	parts := make([]string, 0, len(t.Labels)+1)
	for _, label := range t.Labels {
		parts = append(parts, "["+label+"]")
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
