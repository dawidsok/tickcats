package ticket

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"
)

type Ticket struct {
	Title                 string
	ParsedTitle           ParsedTitle
	Priority              Priority
	Created               time.Time
	Updated               time.Time
	Deadline              *time.Time
	Body                  string
	HasAcceptanceCriteria bool
}

func NewMarkdown(kind Kind, text string, priority Priority, now time.Time, acceptance ...string) string {
	title := ParsedTitle{Kind: kind, Text: strings.TrimSpace(text), HadPrefix: true}.NormalizedTitle()
	return NewMarkdownFull(title, priority, now, acceptance...)
}

func NewMarkdownWithLabels(kind Kind, text string, labels []string, priority Priority, now time.Time, acceptance ...string) string {
	title := ParsedTitle{Kind: kind, Text: strings.TrimSpace(text), Labels: labels, HadPrefix: true}.NormalizedTitle()
	return NewMarkdownFull(title, priority, now, acceptance...)
}

func NewMarkdownFull(fullTitle string, priority Priority, now time.Time, acceptance ...string) string {
	timestamp := now.UTC().Format(time.RFC3339)
	acceptanceText := "-"
	if len(acceptance) > 0 && strings.TrimSpace(acceptance[0]) != "" {
		acceptanceText = "- " + strings.TrimSpace(acceptance[0])
	}

	return fmt.Sprintf(`---
title: %s
priority: %s
created: %s
updated: %s
---

## Context

## Acceptance Criteria
%s
`, fullTitle, priority, timestamp, timestamp, acceptanceText)
}

func ParseMarkdown(data []byte) (Ticket, error) {
	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		return Ticket{}, err
	}

	fields, err := parseFrontmatter(frontmatter)
	if err != nil {
		return Ticket{}, err
	}

	title, err := requiredField(fields, "title")
	if err != nil {
		return Ticket{}, err
	}

	rawPriority, err := requiredField(fields, "priority")
	if err != nil {
		return Ticket{}, err
	}
	priority, err := ParsePriority(rawPriority)
	if err != nil {
		return Ticket{}, err
	}

	created, err := parseRequiredTime(fields, "created")
	if err != nil {
		return Ticket{}, err
	}
	updated, err := parseRequiredTime(fields, "updated")
	if err != nil {
		return Ticket{}, err
	}
	deadline, err := parseOptionalDate(fields, "deadline")
	if err != nil {
		return Ticket{}, err
	}

	bodyText := string(body)
	return Ticket{
		Title:                 title,
		ParsedTitle:           ParseTitle(title),
		Priority:              priority,
		Created:               created,
		Updated:               updated,
		Deadline:              deadline,
		Body:                  bodyText,
		HasAcceptanceCriteria: hasNonEmptySection(bodyText, "Acceptance Criteria"),
	}, nil
}

func splitFrontmatter(data []byte) ([]byte, []byte, error) {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, nil, fmt.Errorf("missing frontmatter opening fence")
	}

	rest := data[len("---\n"):]
	end := bytes.Index(rest, []byte("\n---\n"))
	if end == -1 {
		return nil, nil, fmt.Errorf("missing frontmatter closing fence")
	}

	frontmatter := rest[:end]
	body := rest[end+len("\n---\n"):]
	return frontmatter, body, nil
}

func parseFrontmatter(data []byte) (map[string]string, error) {
	fields := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid frontmatter line %d: %q", lineNumber, line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("invalid frontmatter line %d: empty key", lineNumber)
		}
		fields[key] = strings.Trim(value, `"'`)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return fields, nil
}

func requiredField(fields map[string]string, key string) (string, error) {
	value := strings.TrimSpace(fields[key])
	if value == "" {
		return "", fmt.Errorf("missing required frontmatter field %q", key)
	}
	return value, nil
}

func parseRequiredTime(fields map[string]string, key string) (time.Time, error) {
	value, err := requiredField(fields, key)
	if err != nil {
		return time.Time{}, err
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s timestamp %q: %w", key, value, err)
	}
	return parsed, nil
}

func parseOptionalDate(fields map[string]string, key string) (*time.Time, error) {
	value := strings.TrimSpace(fields[key])
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s date %q: expected YYYY-MM-DD: %w", key, value, err)
	}
	return &parsed, nil
}

func hasNonEmptySection(markdown string, heading string) bool {
	wanted := "## " + heading
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if inSection {
				return false
			}
			inSection = trimmed == wanted
			continue
		}
		if !inSection {
			continue
		}
		if isMeaningfulSectionLine(trimmed) {
			return true
		}
	}

	return false
}

func isMeaningfulSectionLine(line string) bool {
	if line == "" {
		return false
	}
	trimmedBullet := strings.TrimSpace(strings.TrimPrefix(line, "-"))
	return trimmedBullet != ""
}
