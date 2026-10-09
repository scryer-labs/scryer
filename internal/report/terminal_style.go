package report

import (
	"sort"
	"strings"
)

// Semantic styles are presentation only: no status or evidence is inferred here.
type terminalStyle struct{ enabled bool }

func (s terminalStyle) paint(code, value string) string {
	if !s.enabled || value == "" {
		return value
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
}
func (s terminalStyle) heading(value string) string { return s.paint("1;36", value) }
func (s terminalStyle) muted(value string) string   { return s.paint("2", value) }
func (s terminalStyle) warning(value string) string { return s.paint("33", value) }
func (s terminalStyle) status(value string) string {
	switch value {
	case "SUCCEEDED", "PASSED", "EXECUTED", "CURRENT", "RESOLVED", "configured", "present", "complete":
		return s.paint("32", value)
	case "FAILED", "ERROR", "TIMED_OUT", "SNAPSHOT_CHANGED", "NOT_EXECUTED", "UNRESOLVED", "LOOKUP_FAILED":
		return s.paint("31", value)
	case "PARTIALLY_EXECUTED", "UNKNOWN", "UNAVAILABLE", "PARTIAL", "UPDATE_AVAILABLE", "CURRENT_AHEAD", "unknown", "unavailable", "not found", "partial", "UNKNOWN_CURRENT":
		return s.warning(value)
	case "SKIPPED", "not detected":
		return s.muted(value)
	default:
		return value
	}
}
func (s terminalStyle) fact(value string) string {
	// A combined Java source/target row can contain one known and one unknown version.
	parts := strings.Split(value, " / ")
	for i, part := range parts {
		parts[i] = s.status(part)
	}
	return strings.Join(parts, " / ")
}
func (s terminalStyle) counts(values Object) string {
	if len(values) == 0 {
		return s.status("unknown")
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := []string{}
	for _, key := range keys {
		label := s.status(key)
		if text(values[key]) == "0" {
			label = s.muted(key)
		}
		parts = append(parts, label+": "+text(values[key]))
	}
	return strings.Join(parts, ", ")
}
func (s terminalStyle) symbol(value, impact, role string) string {
	if role == "TEST" {
		return s.paint("35", value)
	}
	switch impact {
	case "CHANGED":
		return s.paint("1;31", value)
	case "CALLER":
		return s.paint("34", value)
	default:
		return s.warning(value)
	}
}

// Preserve the existing remote-list comparison palette.
func remoteColor(status string) string {
	switch status {
	case "CURRENT":
		return "32"
	case "UPDATE_AVAILABLE", "CURRENT_AHEAD":
		return "33"
	default:
		return "31"
	}
}
