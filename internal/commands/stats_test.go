package commands

import (
	"testing"
)

func TestParseSections(t *testing.T) {
	tests := []struct {
		name      string
		section   string
		all       bool
		wantCount int
		wantErr   bool
	}{
		{"defaults", "", false, 3, false},
		{"all flag", "", true, 5, false},
		{"single section", "health", false, 1, false},
		{"multiple sections", "summary,health", false, 2, false},
		{"case insensitive", "SUMMARY,Health", false, 2, false},
		{"with spaces", " summary , health ", false, 2, false},
		{"unknown section", "invalid", false, 0, true},
		{"mixed valid and invalid", "summary,bogus", false, 0, true},
		{"empty after split", ",,,", false, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSections(tt.section, tt.all)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != tt.wantCount {
				t.Errorf("got %d sections, want %d: %v", len(result), tt.wantCount, result)
			}
		})
	}

	t.Run("defaults include correct sections", func(t *testing.T) {
		result, _ := parseSections("", false)
		for _, name := range []string{"summary", "overdue", "blocked"} {
			if !result[name] {
				t.Errorf("default sections missing %q", name)
			}
		}
	})

	t.Run("all includes all sections", func(t *testing.T) {
		result, _ := parseSections("", true)
		for _, name := range []string{"summary", "overdue", "blocked", "health", "recent"} {
			if !result[name] {
				t.Errorf("--all missing %q", name)
			}
		}
	})
}
