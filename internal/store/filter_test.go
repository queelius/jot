package store

import (
	"testing"
	"time"

	"github.com/queelius/jot/internal/entry"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"days", "7d", 7 * 24 * time.Hour, false},
		{"weeks", "2w", 14 * 24 * time.Hour, false},
		{"months", "1m", 30 * 24 * time.Hour, false},
		{"years", "1y", 365 * 24 * time.Hour, false},
		{"single day", "1d", 24 * time.Hour, false},
		{"empty", "", 0, false},
		{"go duration", "24h", 24 * time.Hour, false},
		{"uppercase", "7D", 7 * 24 * time.Hour, false},
		{"with spaces", " 7d ", 7 * 24 * time.Hour, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth time.Month
		wantDay   int
	}{
		{"ISO date", "2024-01-15", 2024, 1, 15},
		{"slash date", "2024/01/15", 2024, 1, 15},
		{"today", "today", today.Year(), today.Month(), today.Day()},
		{"yesterday", "yesterday", today.Add(-24 * time.Hour).Year(), today.Add(-24 * time.Hour).Month(), today.Add(-24 * time.Hour).Day()},
		{"tomorrow", "tomorrow", today.Add(24 * time.Hour).Year(), today.Add(24 * time.Hour).Month(), today.Add(24 * time.Hour).Day()},
		{"uppercase TODAY", "TODAY", today.Year(), today.Month(), today.Day()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ParseDate(tt.input)
			if got.Year() != tt.wantYear || got.Month() != tt.wantMonth || got.Day() != tt.wantDay {
				t.Errorf("ParseDate(%q) = %d-%d-%d, want %d-%d-%d",
					tt.input, got.Year(), got.Month(), got.Day(),
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

func TestParseRelativeDate(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"already date", "2024-01-15", "2024-01-15"},
		{"today", "today", today.Format("2006-01-02")},
		{"tomorrow", "tomorrow", today.Add(24 * time.Hour).Format("2006-01-02")},
		{"3 days", "3d", today.Add(3 * 24 * time.Hour).Format("2006-01-02")},
		{"1 week", "1w", today.Add(7 * 24 * time.Hour).Format("2006-01-02")},
		{"2 weeks", "2w", today.Add(14 * 24 * time.Hour).Format("2006-01-02")},
		{"empty", "", ""},
		{"uppercase", "TODAY", today.Format("2006-01-02")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseRelativeDate(tt.input)
			if got != tt.want {
				t.Errorf("ParseRelativeDate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFilter_Apply(t *testing.T) {
	now := time.Now()
	entries := []*entry.Entry{
		{Title: "Task 1", Type: "task", Status: "open", Priority: "high", Tags: []string{"api"}, Created: now},
		{Title: "Task 2", Type: "task", Status: "done", Priority: "low", Tags: []string{"db"}, Created: now.Add(-24 * time.Hour)},
		{Title: "Note 1", Type: "note", Tags: []string{"api", "design"}, Created: now.Add(-48 * time.Hour)},
		{Title: "Idea 1", Type: "idea", Tags: []string{"feature"}, Created: now.Add(-72 * time.Hour)},
	}

	tests := []struct {
		name   string
		filter *Filter
		want   int
	}{
		{"nil filter", nil, 4},
		{"filter by type task", &Filter{Type: "task"}, 2},
		{"filter by type note", &Filter{Type: "note"}, 1},
		{"filter by status open", &Filter{Status: "open"}, 1},
		{"filter by priority high", &Filter{Priority: "high"}, 1},
		{"filter by tag api", &Filter{Tag: "api"}, 2},
		{"filter by tag db", &Filter{Tag: "db"}, 1},
		{"filter with limit", &Filter{Limit: 2}, 2},
		{"combined filters", &Filter{Type: "task", Status: "open"}, 1},
		{"since filter", &Filter{Since: now.Add(-36 * time.Hour)}, 2},
		{"until filter", &Filter{Until: now.Add(-36 * time.Hour)}, 2},
		{"case insensitive type", &Filter{Type: "TASK"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Apply(entries)
			if len(got) != tt.want {
				t.Errorf("Filter.Apply() returned %d entries, want %d", len(got), tt.want)
			}
		})
	}
}

func TestFilter_matches(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		filter *Filter
		entry  *entry.Entry
		want   bool
	}{
		{
			name:   "type match",
			filter: &Filter{Type: "task"},
			entry:  &entry.Entry{Type: "task"},
			want:   true,
		},
		{
			name:   "type mismatch",
			filter: &Filter{Type: "task"},
			entry:  &entry.Entry{Type: "note"},
			want:   false,
		},
		{
			name:   "tag match",
			filter: &Filter{Tag: "api"},
			entry:  &entry.Entry{Tags: []string{"api", "design"}},
			want:   true,
		},
		{
			name:   "tag mismatch",
			filter: &Filter{Tag: "db"},
			entry:  &entry.Entry{Tags: []string{"api", "design"}},
			want:   false,
		},
		{
			name:   "status match case insensitive",
			filter: &Filter{Status: "OPEN"},
			entry:  &entry.Entry{Status: "open"},
			want:   true,
		},
		{
			name:   "since match",
			filter: &Filter{Since: now.Add(-24 * time.Hour)},
			entry:  &entry.Entry{Created: now},
			want:   true,
		},
		{
			name:   "since mismatch",
			filter: &Filter{Since: now},
			entry:  &entry.Entry{Created: now.Add(-24 * time.Hour)},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.matches(tt.entry)
			if got != tt.want {
				t.Errorf("Filter.matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
