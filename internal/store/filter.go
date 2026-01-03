package store

import (
	"strings"
	"time"

	"github.com/queelius/jot/internal/entry"
)

// Filter defines criteria for listing entries.
type Filter struct {
	Type     string
	Tag      string
	Status   string
	Priority string
	Since    time.Time
	Until    time.Time
	Limit    int
}

// Apply filters a slice of entries based on the filter criteria.
func (f *Filter) Apply(entries []*entry.Entry) []*entry.Entry {
	if f == nil {
		return entries
	}

	var result []*entry.Entry

	for _, e := range entries {
		if f.matches(e) {
			result = append(result, e)
		}
	}

	// Apply limit
	if f.Limit > 0 && len(result) > f.Limit {
		result = result[:f.Limit]
	}

	return result
}

func (f *Filter) matches(e *entry.Entry) bool {
	// Type filter
	if f.Type != "" && !strings.EqualFold(e.Type, f.Type) {
		return false
	}

	// Tag filter
	if f.Tag != "" && !e.HasTag(f.Tag) {
		return false
	}

	// Status filter
	if f.Status != "" && !strings.EqualFold(e.Status, f.Status) {
		return false
	}

	// Priority filter
	if f.Priority != "" && !strings.EqualFold(e.Priority, f.Priority) {
		return false
	}

	// Since filter
	if !f.Since.IsZero() && e.Created.Before(f.Since) {
		return false
	}

	// Until filter
	if !f.Until.IsZero() && e.Created.After(f.Until) {
		return false
	}

	return true
}

// ParseDuration parses a duration string like "7d" or "2w" into a time.Duration.
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	// Handle special suffixes
	s = strings.TrimSpace(strings.ToLower(s))

	var multiplier time.Duration
	var numStr string

	switch {
	case strings.HasSuffix(s, "d"):
		multiplier = 24 * time.Hour
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "w"):
		multiplier = 7 * 24 * time.Hour
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "m"):
		multiplier = 30 * 24 * time.Hour
		numStr = s[:len(s)-1]
	case strings.HasSuffix(s, "y"):
		multiplier = 365 * 24 * time.Hour
		numStr = s[:len(s)-1]
	default:
		// Try standard Go duration
		return time.ParseDuration(s)
	}

	var num int
	_, err := parsePositiveInt(numStr, &num)
	if err != nil {
		return 0, err
	}

	return time.Duration(num) * multiplier, nil
}

func parsePositiveInt(s string, result *int) (bool, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		n = n*10 + int(c-'0')
	}
	*result = n
	return true, nil
}

// ParseDate parses a date string in various formats.
func ParseDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"01-02-2006",
		"01/02/2006",
		time.RFC3339,
	}

	// Handle relative dates
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "today":
		return time.Now().Truncate(24 * time.Hour), nil
	case "yesterday":
		return time.Now().Add(-24 * time.Hour).Truncate(24 * time.Hour), nil
	case "tomorrow":
		return time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour), nil
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, nil
}
