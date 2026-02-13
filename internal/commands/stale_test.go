package commands

import (
	"os"
	"testing"
	"time"

	"github.com/queelius/jot/internal/entry"
	"github.com/queelius/jot/internal/store"
)

// TestFormatAge tests the formatAge helper with table-driven tests.
func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"less than a day", 12 * time.Hour, "<1d"},
		{"exactly one day", 24 * time.Hour, "1d"},
		{"3 days", 3 * 24 * time.Hour, "3d"},
		{"6 days", 6 * 24 * time.Hour, "6d"},
		{"7 days (1 week)", 7 * 24 * time.Hour, "1w"},
		{"14 days (2 weeks)", 14 * 24 * time.Hour, "2w"},
		{"20 days", 20 * 24 * time.Hour, "2w"},
		{"29 days", 29 * 24 * time.Hour, "4w"},
		{"30 days (1 month)", 30 * 24 * time.Hour, "1m"},
		{"60 days (2 months)", 60 * 24 * time.Hour, "2m"},
		{"90 days (3 months)", 90 * 24 * time.Hour, "3m"},
		{"180 days (6 months)", 180 * 24 * time.Hour, "6m"},
		{"364 days", 364 * 24 * time.Hour, "12m"},
		{"365 days (1 year)", 365 * 24 * time.Hour, "1y"},
		{"730 days (2 years)", 730 * 24 * time.Hour, "2y"},
		{"zero duration", 0, "<1d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAge(tt.duration)
			if result != tt.expected {
				t.Errorf("formatAge(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

// TestExcludeStatus tests the excludeStatus helper.
func TestExcludeStatus(t *testing.T) {
	entries := []*entry.Entry{
		{Title: "Open task", Status: "open"},
		{Title: "Done task", Status: "done"},
		{Title: "Archived note", Status: "archived"},
		{Title: "In progress", Status: "in_progress"},
		{Title: "Blocked task", Status: "blocked"},
		{Title: "No status", Status: ""},
	}

	t.Run("exclude done and archived", func(t *testing.T) {
		result := excludeStatus(entries, "done", "archived")
		if len(result) != 4 {
			t.Errorf("got %d entries, want 4", len(result))
		}
		for _, e := range result {
			if e.Status == "done" || e.Status == "archived" {
				t.Errorf("entry %q with status %q should have been excluded", e.Title, e.Status)
			}
		}
	})

	t.Run("exclude single status", func(t *testing.T) {
		result := excludeStatus(entries, "open")
		if len(result) != 5 {
			t.Errorf("got %d entries, want 5", len(result))
		}
	})

	t.Run("exclude nothing", func(t *testing.T) {
		result := excludeStatus(entries)
		if len(result) != len(entries) {
			t.Errorf("got %d entries, want %d", len(result), len(entries))
		}
	})

	t.Run("exclude nonexistent status", func(t *testing.T) {
		result := excludeStatus(entries, "nonexistent")
		if len(result) != len(entries) {
			t.Errorf("got %d entries, want %d", len(result), len(entries))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		result := excludeStatus(nil, "done")
		if len(result) != 0 {
			t.Errorf("got %d entries, want 0", len(result))
		}
	})
}

// TestFindStaleEntries tests the findStaleEntries function using the store.
func TestFindStaleEntries(t *testing.T) {
	s, _ := setupTestJournal(t)

	// Create entries with different ages
	old := createTestEntryWithAge(t, s, "Old idea", "idea", "open", "", 120)
	createTestEntryWithAge(t, s, "Recent note", "note", "", "", 10)
	createTestEntryWithAge(t, s, "Old done task", "task", "done", "", 120)
	createTestEntryWithAge(t, s, "Old archived", "note", "archived", "", 200)
	staleTask := createTestEntryWithAge(t, s, "Stale task", "task", "open", "", 100)

	t.Run("default 90-day threshold", func(t *testing.T) {
		entries, err := findStaleEntries(s, 90*24*time.Hour, "", "")
		if err != nil {
			t.Fatalf("findStaleEntries failed: %v", err)
		}
		// Should find: old idea (120d, open) and stale task (100d, open)
		// Should NOT find: recent note (10d), old done task (done), old archived (archived)
		if len(entries) != 2 {
			t.Errorf("got %d entries, want 2", len(entries))
			for _, e := range entries {
				t.Logf("  found: %s (status=%s)", e.Title, e.Status)
			}
		}
		// Stalest first (old idea at 120d before stale task at 100d)
		if len(entries) >= 2 {
			if entries[0].Slug != old.Slug {
				t.Errorf("first entry should be stalest; got %q, want %q", entries[0].Slug, old.Slug)
			}
			if entries[1].Slug != staleTask.Slug {
				t.Errorf("second entry should be %q; got %q", staleTask.Slug, entries[1].Slug)
			}
		}
	})

	t.Run("shorter threshold catches more", func(t *testing.T) {
		entries, err := findStaleEntries(s, 5*24*time.Hour, "", "")
		if err != nil {
			t.Fatalf("findStaleEntries failed: %v", err)
		}
		// All non-done/non-archived entries older than 5 days
		// old idea (120d), recent note (10d), stale task (100d)
		if len(entries) != 3 {
			t.Errorf("got %d entries, want 3", len(entries))
		}
	})

	t.Run("filter by type", func(t *testing.T) {
		entries, err := findStaleEntries(s, 90*24*time.Hour, "task", "")
		if err != nil {
			t.Fatalf("findStaleEntries failed: %v", err)
		}
		// Only stale task (task type, open, 100d old)
		if len(entries) != 1 {
			t.Errorf("got %d entries, want 1", len(entries))
		}
	})

	t.Run("very long threshold finds nothing", func(t *testing.T) {
		entries, err := findStaleEntries(s, 365*24*time.Hour, "", "")
		if err != nil {
			t.Fatalf("findStaleEntries failed: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("got %d entries, want 0", len(entries))
		}
	})
}

// TestFindEntriesOlderThan tests the findEntriesOlderThan function.
func TestFindEntriesOlderThan(t *testing.T) {
	s, _ := setupTestJournal(t)

	createTestEntryWithAge(t, s, "Very old", "idea", "open", "", 200)
	createTestEntryWithAge(t, s, "Somewhat old", "note", "done", "", 120)
	createTestEntryWithAge(t, s, "Recent", "task", "open", "", 10)
	createTestEntryWithAge(t, s, "Old archived", "note", "archived", "", 150)

	t.Run("older than 90d excluding archived", func(t *testing.T) {
		entries, err := findEntriesOlderThan(s, 90*24*time.Hour, "", "", "archived")
		if err != nil {
			t.Fatalf("findEntriesOlderThan failed: %v", err)
		}
		// Should find: very old (200d, open), somewhat old (120d, done)
		// Should NOT find: recent (10d), old archived (excluded)
		if len(entries) != 2 {
			t.Errorf("got %d entries, want 2", len(entries))
		}
	})

	t.Run("no exclusions", func(t *testing.T) {
		entries, err := findEntriesOlderThan(s, 90*24*time.Hour, "", "")
		if err != nil {
			t.Fatalf("findEntriesOlderThan failed: %v", err)
		}
		// Should find: very old, somewhat old, old archived (all >90d)
		if len(entries) != 3 {
			t.Errorf("got %d entries, want 3", len(entries))
		}
	})
}

// TestOutputStaleTable tests that the stale table renders without errors.
func TestOutputStaleTable(t *testing.T) {
	now := time.Now()
	entries := []*entry.Entry{
		{
			Slug:     "20240101-old-idea",
			Title:    "Old Idea",
			Type:     "idea",
			Status:   "open",
			Modified: now.Add(-120 * 24 * time.Hour),
		},
		{
			Slug:     "20240115-stale-task",
			Title:    "Stale Task",
			Type:     "task",
			Status:   "in_progress",
			Modified: now.Add(-45 * 24 * time.Hour),
		},
	}

	output := captureOutput(func() {
		outputStaleTable(entries)
	})

	// Verify table headers
	if output == "" {
		t.Error("output should not be empty")
	}
	if len(output) < 10 {
		t.Errorf("output seems too short: %q", output)
	}
}

// createTestEntryWithAge creates a test entry with its Modified time set to daysAgo days in the past.
func createTestEntryWithAge(t *testing.T, s *store.Store, title, typ, status, tags string, daysAgo int) *entry.Entry {
	t.Helper()
	now := time.Now()
	modified := now.Add(-time.Duration(daysAgo) * 24 * time.Hour)

	e := &entry.Entry{
		Title:    title,
		Type:     typ,
		Status:   status,
		Content:  "Test content for " + title,
		Created:  modified, // Use same time for created to ensure consistent slug generation
		Modified: modified,
	}
	if tags != "" {
		e.Tags = parseTags(tags)
	}
	e.Slug = entry.GenerateSlug(title, e.Created)

	if err := s.Create(e); err != nil {
		t.Fatalf("failed to create test entry %q: %v", title, err)
	}

	// s.Create preserves the struct's Modified value (doesn't call s.Update),
	// but the file on disk has Modified from frontmatter. We need to re-read
	// to confirm, but more importantly the entry we return has the right Modified.
	// However, to ensure the on-disk file also has the correct Modified,
	// we write it directly.
	e.Modified = modified
	content := e.ToMarkdown()
	if err := os.WriteFile(e.Path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write backdated entry: %v", err)
	}

	return e
}
