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
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch s {
	case "today":
		return today, nil
	case "yesterday":
		return today.Add(-24 * time.Hour), nil
	case "tomorrow":
		return today.Add(24 * time.Hour), nil
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, nil
}

// ParseRelativeDate converts relative date strings (3d, 1w, today) to YYYY-MM-DD format.
// If already in date format, returns as-is. Returns empty string on parse failure.
func ParseRelativeDate(s string) string {
	if s == "" {
		return ""
	}

	s = strings.TrimSpace(s)

	// If already looks like a date, return as-is
	if len(s) == 10 && s[4] == '-' && s[7] == '-' {
		return s
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	lower := strings.ToLower(s)

	// Handle named dates
	switch lower {
	case "today":
		return today.Format("2006-01-02")
	case "tomorrow":
		return today.Add(24 * time.Hour).Format("2006-01-02")
	}

	// Handle relative durations: 3d, 1w, 2w
	if dur, err := ParseDuration(lower); err == nil && dur > 0 {
		return today.Add(dur).Format("2006-01-02")
	}

	// Return as-is if we can't parse (let validation catch bad formats)
	return s
}
